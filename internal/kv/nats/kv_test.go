package nats_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
	natsPubSub "github.com/zhulik/fid/internal/pubsub/nats"
)

var _ = Describe("Nats KV", Ordered, func() {
	injector := do.New()

	do.ProvideValue[core.Config](injector, config.Config{})
	do.Provide(injector, natsPubSub.NewClient)

	kv := lo.Must(nats.NewKV(injector))

	BeforeEach(func(ctx SpecContext) {
		lo.Must0(kv.CreateBucket(ctx, "test", 0))

		lo.Must(kv.Create(ctx, "test", "key", []byte("some - value")))
	})

	AfterEach(func(ctx SpecContext) {
		kv.DeleteBucket(ctx, "test") //nolint:errcheck
	})

	Describe("CreateBucket", func() {
		Context("when bucket exists", func() {
			It("does not return an error", func(ctx SpecContext) {
				err := kv.CreateBucket(ctx, "test", 0)

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when bucket does not exists", func() {
			It("creates the bucket", func(ctx SpecContext) {
				err := kv.CreateBucket(ctx, "test2", 0)
				Expect(err).NotTo(HaveOccurred())

				lo.Must0(kv.DeleteBucket(ctx, "test2"))
			})
		})
	})

	Describe("DeleteBucket", func() {
		Context("when bucket exists", func() {
			It("deletes the bucket", func(ctx SpecContext) {
				err := kv.DeleteBucket(ctx, "test")
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when bucket does not exists", func() {
			It("returns an error", func(ctx SpecContext) {
				err := kv.DeleteBucket(ctx, "test2")
				Expect(err).To(MatchError(core.ErrBucketNotFound))
			})
		})
	})

	Describe("Get", func() {
		Context("when key exists", func() {
			It("returns value", func(ctx SpecContext) {
				value, err := kv.Get(ctx, "test", "key")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("some - value")))
			})
		})

		Context("when key does not exists", func() {
			It("returns an error", func(ctx SpecContext) {
				_, err := kv.Get(ctx, "test", "key2")

				Expect(err).To(MatchError(core.ErrKeyNotFound))
			})
		})
	})

	Describe("All", func() {
		Context("when key exists", func() {
			It("returns all values in the bucket", func(ctx SpecContext) {
				list, err := kv.All(ctx, "test")

				Expect(err).ToNot(HaveOccurred())
				Expect(list).To(HaveLen(1))
				Expect(list[0].Key).To(Equal("key"))
				Expect(list[0].Value).To(Equal([]byte("some - value")))
			})
		})
	})

	Describe("Put", func() {
		Context("when key exists", func() {
			It("updates the value", func(ctx SpecContext) {
				err := kv.Put(ctx, "test", "key", []byte("new - value"))

				Expect(err).ToNot(HaveOccurred())
				value, err := kv.Get(ctx, "test", "key")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("new - value")))
			})
		})

		Context("when key does not exists", func() {
			It("creates the value", func(ctx SpecContext) {
				err := kv.Put(ctx, "test", "key2", []byte("new - value"))

				Expect(err).ToNot(HaveOccurred())
				value, err := kv.Get(ctx, "test", "key2")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("new - value")))
			})
		})
	})

	Describe("Create", func() {
		Context("when key exists", func() {
			It("returns an error", func(ctx SpecContext) {
				_, err := kv.Create(ctx, "test", "key", []byte("new - value"))

				Expect(err).To(MatchError(core.ErrKeyExists))
			})
		})

		Context("when key does not exists", func() {
			It("creates the value", func(ctx SpecContext) {
				_, err := kv.Create(ctx, "test", "key2", []byte("new - value"))

				Expect(err).ToNot(HaveOccurred())
				value, err := kv.Get(ctx, "test", "key2")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("new - value")))
			})
		})
	})

	Describe("Delete", func() {
		Context("when key exists", func() {
			It("deletes the key", func(ctx SpecContext) {
				err := kv.Delete(ctx, "test", "key")

				Expect(err).ToNot(HaveOccurred())
				_, err = kv.Get(ctx, "test", "key")

				Expect(err).To(MatchError(core.ErrKeyNotFound))
			})
		})

		Context("when key does not exists", func() {
			It("does not return an error", func(ctx SpecContext) {
				err := kv.Delete(ctx, "test", "key2")

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("WaitCreated", func() {
		Context("when key exists", func() {
			It("returns the value", func(ctx SpecContext) {
				value, err := kv.WaitCreated(ctx, "test", "key")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("some - value")))
			})
		})

		Context("when key does not exists", func() {
			It("waits for the key to be created", func(ctx SpecContext) {
				done := lo.Async0(func() {
					value, err := kv.WaitCreated(ctx, "test", "key3")

					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(Equal([]byte("new - value")))
				})
				time.Sleep(10 * time.Millisecond)

				lo.Must(kv.Create(ctx, "test", "key3", []byte("new - value")))

				Eventually(done).Should(Receive())
			})
		})

		Context("when context timeout reaches", func() {
			It("waits for the key to be created", func(ctx SpecContext) {
				waitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()

				_, err := kv.WaitCreated(waitCtx, "test", "key3")

				Expect(err).To(MatchError(context.DeadlineExceeded))
			})
		})
	})
})
