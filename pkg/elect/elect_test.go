package elect_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/samber/lo"
	"github.com/zhulik/fid/pkg/elect"
)

const (
	bucketName = "test"
	bucketTTL  = 2 * time.Second
	leaderKey  = "leader"
	instanceID = "instanceID"
)

var _ = Describe("Elect", Ordered, func() {
	nc := lo.Must(nats.Connect(nats.DefaultURL))
	js := lo.Must(jetstream.New(nc))

	var jsKV jetstream.KeyValue
	var kv elect.JetStreamKV
	var elector *elect.Elect

	BeforeAll(func(sc SpecContext) {
		jsKV = lo.Must(js.CreateKeyValue(sc, jetstream.KeyValueConfig{
			Bucket: bucketName,
			TTL:    bucketTTL,
		}))

		kv = elect.JetStreamKV{
			KV:  jsKV,
			Ttl: bucketTTL,
		}
		elector = lo.Must(elect.New(kv, leaderKey, instanceID))
	})

	AfterAll(func(sc SpecContext) {
		lo.Must0(js.DeleteKeyValue(sc, bucketName))
	})

	AfterEach(func(sc SpecContext) {
		lo.Must0(jsKV.Purge(sc, leaderKey))
	})

	Describe("Run", func() {
		Context("when no concurrent nominees", func() {
			Context("when the value does not exist", func() {
				It("returns a channel with with won status", func(sc SpecContext) {
					ctx, cancel := context.WithCancel(sc)

					outcomeCh := elector.Start(ctx)

					outcome := <-outcomeCh

					Expect(outcome.Status).To(Equal(elect.Won))

					cancel()

					outcome = <-outcomeCh

					Expect(outcome.Status).To(Equal(elect.Cancelled))
				})
			})

			Context("when the value exists", func() {
				BeforeEach(func(sc SpecContext) {
					lo.Must(kv.Create(sc, leaderKey, []byte(instanceID)))
				})

				It("returns a channel with with lost status", func(sc SpecContext) {
					ctx, cancel := context.WithCancel(sc)

					outcomeCh := elector.Start(ctx)

					outcome := <-outcomeCh

					Expect(outcome.Status).To(Equal(elect.Lost))
					//
					cancel()
					//
					//outcome = <-outcomeCh
					//
					//Expect(outcome.Status).To(Equal(elect.Cancelled))
				})
			})
		})

	})
})
