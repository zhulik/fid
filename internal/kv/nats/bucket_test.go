package nats_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
	"github.com/zhulik/fid/testhelpers"
	"github.com/zhulik/pal"
)

const (
	oneLevelKey = "namespace.key"
	twoLevelKey = "namespace.key.subkey"
	anotherKey  = "another.key"
)

var _ = Describe("Nats KV Bucket", Serial, func() {
	var p *pal.Pal
	var kv nats.KV
	var bucket core.KVBucket

	BeforeEach(func(ctx SpecContext) {
		p = testhelpers.NewPal(ctx)

		lo.Must0(pal.InjectInto(ctx, p, &kv))

		bucket = lo.Must(kv.CreateBucket(ctx, "test"))
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
				Expect(count).To(Equal(4))
			})
		})

		Context("when a filter is passed", func() {
			It("returns all keys in the bucket filtered by specified filters", func(ctx SpecContext) {
				count, err := bucket.Count(ctx, "namespace.>")

				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(2))
			})
		})
	})

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
})
