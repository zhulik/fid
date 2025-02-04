package elect

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	DefaultUpdateMultiplier = 0.75
	DefaultPollMultiplier   = 0.25
)

var ErrInvalidTTL = errors.New("TTL must be configured for the KeyValue bucket")

type Elect struct {
	KV jetstream.KeyValue

	Config Config
}

func New(ctx context.Context, kv jetstream.KeyValue, key string, id string, opts ...Option) (*Elect, error) {
	status, err := kv.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get KV status: %w", err)
	}

	if status.TTL() == 0 {
		return nil, ErrInvalidTTL
	}

	config := &Config{
		Key:            key,
		ID:             id,
		UpdateInterval: time.Duration(float64(status.TTL()) * DefaultUpdateMultiplier),
		PollInterval:   time.Duration(float64(status.TTL()) * DefaultPollMultiplier),
	}

	for _, opt := range opts {
		opt(config)
	}

	return &Elect{
		KV:     kv,
		Config: *config,
	}, nil
}

func (e Elect) Start(ctx context.Context) chan Outcome {
	outcomeCh := make(chan Outcome, 1)

	go e.election(ctx, outcomeCh)

	return outcomeCh
}

func (e Elect) winner(ctx context.Context, seq uint64, outcomeCh chan<- Outcome) {
	outcomeCh <- Outcome{
		Status: Won,
	}

	ticker := time.NewTicker(e.Config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			outcomeCh <- Outcome{
				Status: Cancelled,
				Error:  ctx.Err(),
			}
			close(outcomeCh)

			return
		case <-ticker.C:
			_, err := e.KV.Update(ctx, e.Config.Key, []byte(e.Config.ID), seq)
			if err != nil {
				if errors.Is(err, jetstream.ErrKeyExists) {
					go e.looser(ctx, outcomeCh)

					return
				}
				outcomeCh <- Outcome{
					Status: Error,
					Error:  err,
				}
				close(outcomeCh)

				return
			}
		}
	}
}

func (e Elect) looser(ctx context.Context, outcomeCh chan<- Outcome) {
	outcomeCh <- Outcome{
		Status: Lost,
	}

	ticker := time.NewTicker(e.Config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := e.KV.Get(ctx, e.Config.Key)
			if err == nil {
				// Key still exists, doing nothing
				continue
			}

			if errors.Is(err, jetstream.ErrKeyNotFound) {
				go e.election(ctx, outcomeCh)

				return
			}
		}
	}
}

func (e Elect) election(ctx context.Context, outcomeCh chan<- Outcome) {
	seq, err := e.KV.Create(ctx, e.Config.Key, []byte(e.Config.ID))
	if err == nil {
		go e.winner(ctx, seq, outcomeCh)

		return
	}

	if errors.Is(err, jetstream.ErrKeyExists) {
		go e.looser(ctx, outcomeCh)

		return
	}
	outcomeCh <- Outcome{
		Status: Error,
		Error:  err,
	}
	close(outcomeCh)
}
