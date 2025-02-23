package docker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/json"
)

const (
	BucketNameFunctions = "functions"
)

type Backend struct {
	docker *client.Client
	config core.Config
	logger logrus.FieldLogger
	kv     core.KV
}

func New(injector *do.Injector) (*Backend, error) {
	return &Backend{
		docker: do.MustInvoke[*client.Client](injector),
		config: do.MustInvoke[core.Config](injector),
		logger: do.MustInvoke[logrus.FieldLogger](injector).WithField("component", "backends.dockerexternal.Backend"),
		kv:     do.MustInvoke[core.KV](injector),
	}, nil
}

// Register creates a new function template container, scaler, forwarder(TODO) and garbage collector(TODO).
func (b Backend) Register(ctx context.Context, function core.Function) error {
	err := b.createFunctionTemplate(ctx, function)
	if err != nil {
		return err
	}

	err = b.createScaler(ctx, function)
	if err != nil {
		return err
	}

	return nil
}

func (b Backend) createScaler(ctx context.Context, function core.Function) error {
	logger := b.logger.WithField("function", function.Name())

	containerConfig := &container.Config{
		Image: core.ImageNameScaler,
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameFunctionName: function.Name(),
			core.EnvNameNatsURL:      b.config.NatsURL(),
		}),
		Labels: map[string]string{
			core.LabelNameComponent: core.ScalerComponentLabelValue,
		},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock", // TODO: configurable
		},
		AutoRemove: true,
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"nats": {}, // TODO: get from config
		},
	}

	containerName := function.Name() + "-scaler"

	_, err := b.docker.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict. The container name") {
			logger.Infof("Scaler container already exists")

			return nil
		}

		return fmt.Errorf("failed to create scaler container: %w", err)
	}

	err = b.docker.ContainerStart(ctx, containerName, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start scaler container: %w", err)
	}

	logger.Infof("Scaler container created and started")

	return nil
}

func (b Backend) createFunctionTemplate(ctx context.Context, function core.Function) error {
	backendFunction := Function{
		Name_:    function.Name(),
		Image_:   function.Image(),
		Timeout_: function.Timeout(),
		MinScale: function.ScalingConfig().Min,
		MaxScale: function.ScalingConfig().Max,
		Env_:     function.Env(),
	}

	bytes, err := json.Marshal(backendFunction)
	if err != nil {
		return fmt.Errorf("failed to marshal function: %w", err)
	}

	err = b.kv.CreateBucket(ctx, BucketNameFunctions, 0)
	if err != nil {
		return fmt.Errorf("failed to create functions bucket: %w", err)
	}

	err = b.kv.Put(ctx, BucketNameFunctions, function.Name(), bytes)
	if err != nil {
		return fmt.Errorf("failed to store function template: %w", err)
	}

	b.logger.WithField("function", function.Name()).Info("Function template stored")

	return nil
}

func (b Backend) Info(ctx context.Context) (map[string]any, error) {
	info, err := b.docker.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to docker info: %w", err)
	}

	return map[string]any{
		"backend":      "Docker backend",
		"dockerEngine": info,
	}, nil
}

func (b Backend) Function(ctx context.Context, name string) (core.Function, error) {
	b.logger.WithField("function", name).Debug("Fetching function info")

	bytes, err := b.kv.Get(ctx, BucketNameFunctions, name)
	if err != nil {
		if errors.Is(err, core.ErrKeyNotFound) {
			return nil, core.ErrFunctionNotFound
		}

		return nil, fmt.Errorf("failed to get function template: %w", err)
	}

	function, err := json.Unmarshal[Function](bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal function template: %w", err)
	}

	return function, nil
}

func (b Backend) Functions(ctx context.Context) ([]core.Function, error) {
	b.logger.Debug("Fetching function list")

	fns, err := b.kv.All(ctx, BucketNameFunctions)
	if err != nil {
		return nil, fmt.Errorf("failed to get function list: %w", err)
	}

	functions := make([]core.Function, len(fns))

	for i, fn := range fns {
		function, err := json.Unmarshal[Function](fn.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal function: %w", err)
		}

		functions[i] = function
	}

	return functions, nil
}

func (b Backend) HealthCheck() error {
	b.logger.Debug("ContainerBackend health check.")

	_, err := b.docker.Info(context.Background())
	if err != nil {
		return fmt.Errorf("backend health check failed: %w", err)
	}

	return nil
}

func (b Backend) Shutdown() error {
	b.logger.Debug("ContainerBackend shutting down...")
	defer b.logger.Debug("ContainerBackend shot down.")

	err := b.docker.Close()
	if err != nil {
		return fmt.Errorf("failed to shut down the backend: %w", err)
	}

	return nil
}

func (b Backend) AddInstance(ctx context.Context, function core.Function) (string, error) {
	b.logger.Infof("Creating new function pod for function %s", function.Name())

	pod, err := CreateFunctionPod(ctx, function)
	if err != nil {
		return "", err
	}

	b.logger.Infof("Function pod function %s created id=%s", function.Name(), pod.UUID)

	return pod.UUID, nil
}

func (b Backend) KillInstance(ctx context.Context, instanceID string) error {
	b.logger.Infof("Killing function instance %s", instanceID)

	return FunctionPod{UUID: instanceID, docker: b.docker}.Delete(ctx)
}

func (b Backend) StartGateway(ctx context.Context) (string, error) {
	// TODO implement me
	panic("implement me")
}

func (b Backend) StartInfoServer(ctx context.Context) (string, error) {
	containerConfig := &container.Config{
		Image: core.ImageNameInfoServer,
		Env: core.MapToEnvList(map[string]string{
			core.EnvNameNatsURL: b.config.NatsURL(),
		}),
		Labels: map[string]string{
			core.LabelNameComponent: core.InfoServerComponentLabelValue,
		},
		ExposedPorts: nat.PortSet{
			"80/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			"/var/run/docker.sock:/var/run/docker.sock", // TODO: configurable
		},
		AutoRemove: true,
		PortBindings: nat.PortMap{
			// TODO: configurable
			"80/tcp": {
				{
					HostPort: "8080",
					HostIP:   "0.0.0.0",
				},
			},
		},
	}
	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"nats": {}, // TODO: get from config
		},
	}

	resp, err := b.docker.ContainerCreate(
		ctx, containerConfig, hostConfig,
		networkingConfig, nil, core.ContainerNameInfoServer,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Conflict. The container name") {
			b.logger.Infof("Info server container already exists")

			return "", core.ErrContainerAlreadyExists
		}

		return "", fmt.Errorf("failed to create info server container: %w", err)
	}

	err = b.docker.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start info server container: %w", err)
	}

	b.logger.Info("Info server container created and started")

	return resp.ID, nil
}
