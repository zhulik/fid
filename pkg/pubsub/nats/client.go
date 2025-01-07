package nats

import (
	"context"
	"fmt"

	libNats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/core"
)

type Client struct {
	nats      *libNats.Conn
	jetStream jetstream.JetStream
}

func (c Client) HealthCheck() error {
	_, err := c.nats.GetClientID()
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	_, err = c.jetStream.AccountInfo(context.Background())
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	return nil
}

func (c Client) Shutdown() error {
	c.jetStream.CleanupPublisher()
	c.nats.Close()

	return nil
}

func NewClient(injector *do.Injector) (*Client, error) {
	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	natsClient, err := libNats.Connect(config.NatsURL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS client: %w", err)
	}

	jetStream, err := jetstream.New(natsClient)
	if err != nil {
		return nil, fmt.Errorf("failed to build jetStream client: %w", err)
	}

	return &Client{
		nats:      natsClient,
		jetStream: jetStream,
	}, nil
}
