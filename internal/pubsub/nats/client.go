package nats

import (
	"context"
	"fmt"
	"time"

	libNats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/config"
)

func NewClient(injector do.Injector) (*Client, error) {
	config := do.MustInvoke[config.Config](injector)

	natsClient, err := libNats.Connect(config.NATSURL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	jetStream, err := jetstream.New(natsClient)
	if err != nil {
		return nil, fmt.Errorf("failed to build JetStream client: %w", err)
	}

	return &Client{
		Nats:      natsClient,
		JetStream: jetStream,
	}, nil
}

type Client struct {
	Nats      *libNats.Conn
	JetStream jetstream.JetStream
	KV        libNats.KeyValue
}

func (c Client) HealthCheck() error {
	_, err := c.Nats.GetClientID()
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.JetStream.AccountInfo(ctx)
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	return nil
}

func (c Client) Shutdown() error {
	c.JetStream.CleanupPublisher()
	c.Nats.Close()

	return nil
}
