package nats

import (
	"context"
	"fmt"

	libNats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
)

func NewClient(injector *do.Injector) (*Client, error) {
	config := do.MustInvoke[core.Config](injector)

	natsClient, err := libNats.Connect(config.NATSURL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS client: %w", err)
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

	_, err = c.JetStream.AccountInfo(context.Background())
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
