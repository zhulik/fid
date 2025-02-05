package elect

import (
	"bytes"
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

// TODO: use a custom interface for kv
// TODO: use timeouts when calling kv methods
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
		ID:             []byte(id),
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
			status, seq, err = e.tick(ctx, status, seq)
			if err != nil {
				status = Error
			}
		default:
			switch status {
			case Unknown:
				ticker.Stop()

				seq, err = e.KV.Create(ctx, e.Config.Key, e.Config.ID)
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

func (e Elect) tick(ctx context.Context, status ElectionStatus, seq uint64) (ElectionStatus, uint64, error) {
	var err error

	entry, err := e.KV.Get(ctx, e.Config.Key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			err = nil
		}

		return Unknown, 0, err
	}

	switch status {
	case Won:
		if !bytes.Equal(entry.Value(), e.Config.ID) {
			// someone else is the leader
			return Unknown, 0, nil
		}

		seq, err = e.KV.Update(ctx, e.Config.Key, e.Config.ID, seq)
	case Lost:
		// do nothing, we performed the check in the beginning
	case Unknown, Error, Cancelled:
		panic(fmt.Sprintf("unexpected status: %v, ticker must have been stopped", status))
	}

	return status, seq, err //nolint:wrapcheck
}
