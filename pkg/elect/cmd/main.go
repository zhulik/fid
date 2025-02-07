package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/lo"
)

const (
	bucketName = "leader-election"
	bucketTTL  = 5 * time.Second
)

var (
	nc = lo.Must(nats.Connect(nats.DefaultURL)) //nolint:gochecknoglobals
	js = lo.Must(jetstream.New(nc))             //nolint:gochecknoglobals,varnamelen
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kv := lo.Must(js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: bucketName,
		TTL:    bucketTTL,
	}))

	log.Printf("Bucket created: %s", bucketName)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	instanceID := uuid.NewString()

	wg := &sync.WaitGroup{}

	go func() {
		started := false

		for {
			sig := <-signalChannel

			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				log.Println("Shutting down...")
				cancel()

				return
			case syscall.SIGUSR1:
				if started {
					log.Println("Election process already started")

					continue
				}

				log.Println("Starting election process...")
				wg.Add(1)

				go runElection(ctx, instanceID, kv, bucketTTL, wg)

				started = true
			}
		}
	}()

	if len(os.Args) > 1 {
		if os.Args[1] == "now" {
			signalChannel <- syscall.SIGUSR1
		}
	}

	log.Printf("Waiting for USR1 signal to start the election process, this instance instanceID is %s", instanceID)

	<-ctx.Done()
	wg.Wait()
}
