package elect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	DefaultTimeout          = 200 * time.Millisecond
	DefaultUpdateMultiplier = 0.75
	DefaultPollMultiplier   = 0.25
)

var ErrInvalidTTL = errors.New("TTL must be configured for the KeyValue bucket")

type Elect struct {
	KV KV

	Config Config
}

func New(kv KV, key string, id string, opts ...Option) (*Elect, error) {
	if kv.TTL() == 0 {
		return nil, ErrInvalidTTL
	}

	config := &Config{
		Key:            key,
		ID:             []byte(id),
		Timeout:        DefaultTimeout,
		UpdateInterval: time.Duration(float64(kv.TTL()) * DefaultUpdateMultiplier),
		PollInterval:   time.Duration(float64(kv.TTL()) * DefaultPollMultiplier),
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
	var currentStatus ElectionStatus

	newStatusCh := make(chan ElectionStatus, 1)

	var seq uint64

	var err error

	ticker := time.NewTicker(e.Config.PollInterval)

	defer close(outcomeCh)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			newStatusCh <- Cancelled
		case <-ticker.C:
			var status ElectionStatus

			status, seq, err = e.tick(ctx, currentStatus, seq)
			if err != nil {
				newStatusCh <- Error

				continue
			}
			newStatusCh <- status
		case newStatus := <-newStatusCh:
			currentStatus = newStatus

			switch newStatus {
			case Unknown:
				ticker.Stop()

				createCtx, cancel := context.WithTimeout(ctx, e.Config.Timeout)

				seq, err = e.KV.Create(createCtx, e.Config.Key, e.Config.ID)

				cancel()

				if err == nil {
					newStatusCh <- Won

					continue
				}

				if errors.Is(err, ErrKeyExists) {
					newStatusCh <- Lost
				} else {
					newStatusCh <- Error
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

	getCtx, cancel := context.WithTimeout(ctx, e.Config.Timeout)
	defer cancel()

	leaderID, err := e.KV.Get(getCtx, e.Config.Key)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			err = nil
		}

		return Unknown, 0, err
	}

	switch status {
	case Won:
		if !bytes.Equal(leaderID, e.Config.ID) {
			// someone else is the leader
			return Unknown, 0, nil
		}

		updateCtx, cancel := context.WithTimeout(ctx, e.Config.Timeout)
		defer cancel()

		seq, err = e.KV.Update(updateCtx, e.Config.Key, e.Config.ID, seq)
	case Lost:
		// do nothing, we performed the check in the beginning
	case Unknown, Error, Cancelled:
		panic(fmt.Sprintf("unexpected status: %v, ticker must have been stopped", status))
	}

	return status, seq, err //nolint:wrapcheck
}
