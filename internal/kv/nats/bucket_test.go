package nats_test

import (
	"math/rand/v2"
	"strconv"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
	"github.com/zhulik/fid/testhelpers"
	"go.uber.org/atomic"
)

const (
	concurrentUpdates = 100
	updateIterations  = 100

	oneLevelKey = "namespace.key"
	twoLevelKey = "namespace.key.subkey"
	anotherKey  = "another.key"
)

var _ = Describe("Nats KV Bucket", Serial, func() {
	var injector *do.Injector
	var kv core.KV

	var bucket core.KVBucket

	BeforeEach(func(ctx SpecContext) {
		injector = testhelpers.NewInjector()

		kv = lo.Must(nats.NewKV(injector))

		bucket = lo.Must(kv.CreateBucket(ctx, "test", 0))
		DeferCleanup(func(ctx SpecContext) { kv.DeleteBucket(ctx, "test") }) //nolint:errcheck

		lo.Must(bucket.Create(ctx, "key", []byte("some - value")))
		lo.Must(bucket.Create(ctx, oneLevelKey, []byte("some - value")))
		lo.Must(bucket.Create(ctx, twoLevelKey, []byte("some - value")))
		lo.Must(bucket.Create(ctx, anotherKey, []byte("some - value")))
	})

	Describe("Get", func() {
		Context("when key exists", func() {
			It("returns value", func(ctx SpecContext) {
				value, err := bucket.Get(ctx, "key")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("some - value")))
			})
		})

		Context("when key does not exists", func() {
			It("returns an error", func(ctx SpecContext) {
				_, err := bucket.Get(ctx, "key2")

				Expect(err).To(MatchError(core.ErrKeyNotFound))
			})
		})
	})

	Describe("Keys", func() {
		Context("when no filters passed", func() {
			It("returns all keys in the bucket", func(ctx SpecContext) {
				keys, err := bucket.Keys(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(keys).To(ConsistOf([]string{
					"key",
					oneLevelKey,
					twoLevelKey,
					anotherKey,
				}))
			})
		})

		Context("when a filter is passed", func() {
			It("returns all keys in the bucket filtered by specified filters", func(ctx SpecContext) {
				keys, err := bucket.Keys(ctx, "namespace.>")
				Expect(err).ToNot(HaveOccurred())
				Expect(keys).To(ConsistOf([]string{
					oneLevelKey,
					twoLevelKey,
				}))
			})
		})
	})

	Describe("Count", func() {
		Context("when no filters passed", func() {
			It("returns all keys in the bucket", func(ctx SpecContext) {
				count, err := bucket.Count(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(int64(4)))
			})
		})

		Context("when a filter is passed", func() {
			It("returns all keys in the bucket filtered by specified filters", func(ctx SpecContext) {
				count, err := bucket.Count(ctx, "namespace.>")
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(int64(2)))
			})
		})
	})

	Describe("Count", func() {})

	Describe("All", func() {
		Context("when no filters passed", func() {
			It("returns all values in the bucket", func(ctx SpecContext) {
				list, err := bucket.All(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(list).To(HaveLen(4))
				Expect(list[0].Key).To(Equal("key"))
				Expect(list[0].Value).To(Equal([]byte("some - value")))
			})
		})

		Context("when a full key specified", func() {
			It("returns all values in the bucket filtered by specified filters", func(ctx SpecContext) {
				list, err := bucket.All(ctx, "key")

				Expect(err).ToNot(HaveOccurred())
				Expect(list).To(HaveLen(1))
				Expect(list[0].Key).To(Equal("key"))
				Expect(list[0].Value).To(Equal([]byte("some - value")))
			})
		})

		Context("when a wildcards are used", func() {
			Context("when * is used", func() {
				It("returns all values in the bucket filtered by specified filters", func(ctx SpecContext) {
					list, err := bucket.All(ctx, "namespace.*")

					Expect(err).ToNot(HaveOccurred())
					Expect(list).To(HaveLen(1))
					Expect(list[0].Key).To(Equal(oneLevelKey))
					Expect(list[0].Value).To(Equal([]byte("some - value")))
				})
			})

			Context("when > is used", func() {
				It("returns all values in the bucket filtered by specified filters", func(ctx SpecContext) {
					list, err := bucket.All(ctx, "namespace.>")

					Expect(err).ToNot(HaveOccurred())
					Expect(list).To(HaveLen(2))
					Expect(list[0].Key).To(Equal(oneLevelKey))
					Expect(list[0].Value).To(Equal([]byte("some - value")))

					Expect(list[1].Key).To(Equal(twoLevelKey))
					Expect(list[1].Value).To(Equal([]byte("some - value")))
				})
			})

			Context("when multiple filters are used", func() {
				It("returns all values in the bucket filtered by specified filters", func(ctx SpecContext) {
					list, err := bucket.All(ctx, "namespace.>", "another.*")

					Expect(err).ToNot(HaveOccurred())
					Expect(list).To(HaveLen(3))
					Expect(list[0].Key).To(Equal(oneLevelKey))
					Expect(list[0].Value).To(Equal([]byte("some - value")))

					Expect(list[1].Key).To(Equal(twoLevelKey))
					Expect(list[1].Value).To(Equal([]byte("some - value")))

					Expect(list[2].Key).To(Equal(anotherKey))
					Expect(list[2].Value).To(Equal([]byte("some - value")))
				})
			})
		})
	})

	Describe("Put", func() {
		Context("when key exists", func() {
			It("updates the value", func(ctx SpecContext) {
				err := bucket.Put(ctx, "key", []byte("new - value"))

				Expect(err).ToNot(HaveOccurred())
				value, err := bucket.Get(ctx, "key")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("new - value")))
			})
		})

		Context("when key does not exists", func() {
			It("creates the value", func(ctx SpecContext) {
				err := bucket.Put(ctx, "key2", []byte("new - value"))

				Expect(err).ToNot(HaveOccurred())
				value, err := bucket.Get(ctx, "key2")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("new - value")))
			})
		})
	})

	Describe("Upsert", func() {
		Context("when key exists", func() {
			It("returns an error", func(ctx SpecContext) {
				_, err := bucket.Create(ctx, "key", []byte("new - value"))

				Expect(err).To(MatchError(core.ErrKeyExists))
			})
		})

		Context("when key does not exists", func() {
			It("creates the value", func(ctx SpecContext) {
				_, err := bucket.Create(ctx, "key2", []byte("new - value"))

				Expect(err).ToNot(HaveOccurred())
				value, err := bucket.Get(ctx, "key2")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("new - value")))
			})
		})
	})

	Describe("Delete", func() {
		Context("when key exists", func() {
			It("deletes the key", func(ctx SpecContext) {
				err := bucket.Delete(ctx, "key")

				Expect(err).ToNot(HaveOccurred())
				_, err = bucket.Get(ctx, "key")

				Expect(err).To(MatchError(core.ErrKeyNotFound))
			})
		})

		Context("when key does not exists", func() {
			It("does not return an error", func(ctx SpecContext) {
				err := bucket.Delete(ctx, "key2")

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Incr", func() {
		Context("when counter does not exist", func() {
			It("creates the counter", func(ctx SpecContext) {
				value, err := bucket.Incr(ctx, "counter", 5)

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal(int64(5)))
			})
		})

		Context("when counter exists", func() {
			Context("when no concurrent updates", func() {
				It("increments the counter", func(ctx SpecContext) {
					lo.Must(bucket.Incr(ctx, "counter", 5))

					value, err := bucket.Incr(ctx, "counter", 3)

					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(Equal(int64(8)))
				})
			})

			Context("when concurrent updates", func() {
				It("increments the counter", func(ctx SpecContext) {
					sum := atomic.NewInt64(0)

					wg := sync.WaitGroup{}
					wg.Add(concurrentUpdates)

					for range concurrentUpdates {
						go func() {
							defer wg.Done()

							for range updateIterations {
								n := rand.Int64() //nolint:gosec
								_, err := bucket.Incr(ctx, "counter", n)
								Expect(err).ToNot(HaveOccurred())
								sum.Add(n)
							}
						}()
					}

					wg.Wait()

					bytes, err := bucket.Get(ctx, "counter")
					Expect(err).ToNot(HaveOccurred())

					value, err := strconv.ParseInt(string(bytes), 10, 64)
					Expect(err).ToNot(HaveOccurred())

					Expect(value).To(Equal(sum.Load()))
				})
			})
		})
	})
})
