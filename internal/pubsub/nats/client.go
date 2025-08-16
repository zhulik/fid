package nats

import (
	"context"
	"fmt"
	"time"

	libNats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/zhulik/fid/internal/config"
)

type Client struct { //nolint:recvcheck
	Config    *config.Config
	Nats      *libNats.Conn
	JetStream jetstream.JetStream
}

func (c *Client) Init(_ context.Context) error {
	natsClient, err := libNats.Connect(c.Config.NATSURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	jetStream, err := jetstream.New(natsClient)
	if err != nil {
		return fmt.Errorf("failed to build JetStream client: %w", err)
	}

	c.Nats = natsClient
	c.JetStream = jetStream

	return nil
}

func (c Client) HealthCheck(ctx context.Context) error {
	_, err := c.Nats.GetClientID()
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	_, err = c.JetStream.AccountInfo(ctx)
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	return nil
}

func (c Client) Shutdown(_ context.Context) error {
	c.JetStream.CleanupPublisher()
	c.Nats.Close()

	return nil
}
