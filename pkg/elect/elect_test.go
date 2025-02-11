package elect_test

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/zhulik/fid/pkg/elect"
)

const (
	bucketName = "test"
	bucketTTL  = 2 * time.Second
	leaderKey  = "leader"
	instanceID = "instanceID"
)

var (
	nc = lo.Must(nats.Connect(nats.DefaultURL))
	js = lo.Must(jetstream.New(nc))
)

var _ = Describe("Elect", Ordered, func() {
	Describe(".Start", func() {
		var jsKV jetstream.KeyValue
		var kv elect.JetStreamKV //nolint:varnamelen
		var elector *elect.Elect

		BeforeAll(func(sctx SpecContext) {
			jsKV = lo.Must(js.CreateKeyValue(sctx, jetstream.KeyValueConfig{
				Bucket: bucketName,
				TTL:    bucketTTL,
			}))

			kv = elect.JetStreamKV{
				KV:  jsKV,
				Ttl: bucketTTL,
			}
			elector = lo.Must(elect.New(kv, leaderKey, instanceID))
		})

		AfterAll(func(sctx SpecContext) {
			lo.Must0(js.DeleteKeyValue(sctx, bucketName))
		})

		AfterEach(func(sctx SpecContext) {
			lo.Must0(jsKV.Purge(sctx, leaderKey))
		})

		Describe("Run", func() {
			Context("when no concurrent nominees", func() {
				Context("when the value does not exist", func() {
					It("returns a channel with won status", func(sctx SpecContext) {
						ctx, cancel := context.WithCancel(sctx)

						outcomeCh := elector.Start(ctx)

						outcome := <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Won))

						cancel()

						outcome = <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Cancelled))
					})

					It("keeps the record updated", func(sctx SpecContext) {
						ctx, cancel := context.WithCancel(sctx)

						outcomeCh := elector.Start(ctx)

						outcome := <-outcomeCh

						entry := lo.Must(jsKV.Get(ctx, leaderKey))

						Expect(outcome.Status).To(Equal(elect.Won))

						revision := entry.Revision()

						time.Sleep(bucketTTL)

						entry = lo.Must(jsKV.Get(ctx, leaderKey))
						Expect(entry.Revision()).To(Equal(revision + 1))

						cancel()

						outcome = <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Cancelled))
					})

					It("becomes a looser if the value changes", func(sctx SpecContext) {
						ctx, cancel := context.WithCancel(sctx)

						outcomeCh := elector.Start(ctx)

						outcome := <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Won))

						lo.Must(jsKV.Put(sctx, leaderKey, []byte("anotherInstanceID")))

						outcome = <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Lost))

						cancel()

						outcome = <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Cancelled))
					})
				})

				Context("when the value exists", func() {
					BeforeEach(func(sctx SpecContext) {
						lo.Must(kv.Create(sctx, leaderKey, []byte("anotherInstanceID")))
					})

					It("returns a channel with lost status", func(sctx SpecContext) {
						ctx, cancel := context.WithCancel(sctx)

						outcomeCh := elector.Start(ctx)

						outcome := <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Lost))

						cancel()

						outcome = <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Cancelled))
					})

					It("becomes a leader if the value is deleted", func(sctx SpecContext) {
						ctx, cancel := context.WithCancel(sctx)

						outcomeCh := elector.Start(ctx)

						outcome := <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Lost))

						lo.Must0(jsKV.Purge(sctx, leaderKey))

						outcome = <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Won))

						cancel()

						outcome = <-outcomeCh

						Expect(outcome.Status).To(Equal(elect.Cancelled))
					})
				})
			})
		})
	})
})
