package elect_test

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/zhulik/fid/pkg/elect"
)

const (
	bucketName   = "test"
	bucketTTL    = 2 * time.Second
	leaderKey    = "leader"
	instanceID   = "instanceID"
	nomineeCount = 100
)

var (
	nc = lo.Must(nats.Connect(nats.DefaultURL))
	js = lo.Must(jetstream.New(nc))
)

type Nomenee struct {
	InstanceID string
	Elect      *elect.Elect

	Cancel context.CancelFunc

	ctx    context.Context //nolint:containedctx
	status atomic.Int32
}

func newNomenee(ctx context.Context, kv jetstream.KeyValue) *Nomenee {
	elector := lo.Must(elect.New(elect.JetStreamKV{
		KV:  kv,
		Ttl: bucketTTL,
	}, leaderKey, uuid.NewString()))

	ctx, cancel := context.WithCancel(ctx)

	return &Nomenee{
		InstanceID: instanceID,
		Elect:      elector,
		Cancel:     cancel,
		ctx:        ctx,
		status:     atomic.Int32{},
	}
}

func (n *Nomenee) Status() elect.ElectionStatus {
	return elect.ElectionStatus(n.status.Load())
}

func (n *Nomenee) Run() {
	for outcome := range n.Elect.Start(n.ctx) {
		n.status.Store(int32(outcome.Status))

		switch outcome.Status { //nolint:exhaustive
		// case elect.Won:
		//	log.Printf("Election won, I'm the leader")
		// case elect.Lost:
		//	log.Printf("Election lost, someone else is the leader")
		// case elect.Error:
		//	log.Printf("Error: %s", outcome.Error)
		// return
		case elect.Cancelled:
			return
		case elect.Unknown:
			panic("unexpected status")
		}
	}
}

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

			Context("when multiple concurrent nominees", func() {
				It("elects only one leader", NodeTimeout(20*time.Second), func(sctx SpecContext) {
					nominees := make([]*Nomenee, nomineeCount)

					for i := range nomineeCount {
						nominee := newNomenee(sctx, jsKV)
						nominees[i] = nominee

						go nominee.Run()
					}
					time.Sleep(100 * time.Millisecond)

					for range 5 {
						leaders := lo.Filter(nominees, func(n *Nomenee, _ int) bool {
							return n.Status() == elect.Won
						})

						Expect(leaders).To(HaveLen(1))

						leaders[0].Cancel()

						time.Sleep(time.Duration(float64(bucketTTL) * 1.5))
					}

					for _, nominee := range nominees {
						nominee.Cancel()
					}

					time.Sleep(100 * time.Millisecond)

					cancelled := lo.Filter(nominees, func(n *Nomenee, _ int) bool {
						return n.Status() == elect.Cancelled
					})

					Expect(cancelled).To(HaveLen(nomineeCount))
				})
			})
		})
	})
})
