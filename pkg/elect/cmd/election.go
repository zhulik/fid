package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/lo"
	"github.com/zhulik/fid/pkg/elect"
)

const (
	leaderKey = "leader"
)

func runElection(ctx context.Context, instanceID string, kv jetstream.KeyValue, ttl time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()

	kvWrapper := elect.JetStreamKV{
		KV:  kv,
		Ttl: ttl,
	}

	el := lo.Must(elect.New(kvWrapper, leaderKey, instanceID))

	outcomeCh := el.Start(ctx)

	for outcome := range outcomeCh {
		switch outcome.Status {
		case elect.Won:
			log.Printf("Election won, I'm the leader")
		case elect.Lost:
			log.Printf("Election lost, someone else is the leader")
		case elect.Error:
			log.Printf("Error: %s", outcome.Error)

			return
		case elect.Cancelled:
			log.Println("Election stopped")

			return
		case elect.Unknown:
			panic("unexpected status")
		}
	}
}
