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

func (e Elect) election(ctx context.Context, outcomeCh chan<- Outcome) { //nolint:cyclop,funlen
	var status ElectionStatus

	var seq uint64

	var err error

	ticker := time.NewTicker(e.Config.PollInterval)

	defer close(outcomeCh)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			status = Cancelled
		case <-ticker.C:
			switch status {
			case Won:
				seq, err = e.KV.Update(ctx, e.Config.Key, []byte(e.Config.ID), seq)
				if errors.Is(err, jetstream.ErrKeyExists) {
					status = Lost
				}
			case Lost:
				_, err = e.KV.Get(ctx, e.Config.Key)
				if errors.Is(err, jetstream.ErrKeyNotFound) {
					status = Unknown

					continue
				}

				status = Error
			case Unknown, Error, Cancelled:
				panic(fmt.Sprintf("unexpected status: %v", status))
			}
		default:
			switch status {
			case Unknown:
				ticker.Stop()

				seq, err = e.KV.Create(ctx, e.Config.Key, []byte(e.Config.ID))
				if err == nil {
					status = Won

					continue
				}

				if errors.Is(err, jetstream.ErrKeyExists) {
					status = Lost
				} else {
					status = Error
				}
			case Won:
				outcomeCh <- Outcome{
					Status: Won,
				}
				ticker.Reset(e.Config.UpdateInterval)
			case Lost:
				outcomeCh <- Outcome{
					Status: Lost,
				}
				ticker.Reset(e.Config.PollInterval)
			case Error:
				outcomeCh <- Outcome{
					Status: Error,
					Error:  err,
				}

				return
			case Cancelled:
				outcomeCh <- Outcome{
					Status: Cancelled,
					Error:  ctx.Err(),
				}

				return
			}
		}
	}
}
