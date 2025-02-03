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
	ctx := context.Background()

	do.ProvideValue[core.Config](injector, config.Config{})
	do.Provide(injector, natsPubSub.NewClient)

	kv := lo.Must(nats.NewKV(injector))

	BeforeEach(func() {
		lo.Must0(kv.CreateBucket(context.Background(), "test"))

		lo.Must0(kv.Create(ctx, "test", "key", []byte("some - value")))
	})

	AfterEach(func() {
		kv.DeleteBucket(context.Background(), "test") //nolint:errcheck
	})

	Describe("CreateBucket", func() {
		Context("when bucket exists", func() {
			It("does not return an error", func() {
				err := kv.CreateBucket(context.Background(), "test")

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when bucket does not exists", func() {
			It("creates the bucket", func() {
				err := kv.CreateBucket(context.Background(), "test2")
				Expect(err).NotTo(HaveOccurred())

				lo.Must0(kv.DeleteBucket(ctx, "test2"))
			})
		})
	})

	Describe("DeleteBucket", func() {
		Context("when bucket exists", func() {
			It("deletes the bucket", func() {
				err := kv.DeleteBucket(context.Background(), "test")
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when bucket does not exists", func() {
			It("returns an error", func() {
				err := kv.DeleteBucket(context.Background(), "test2")
				Expect(err).To(MatchError(core.ErrBucketNotFound))
			})
		})
	})

	Describe("Get", func() {
		Context("when key exists", func() {
			It("returns value", func() {
				value, err := kv.Get(ctx, "test", "key")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("some - value")))
			})
		})

		Context("when key does not exists", func() {
			It("returns an error", func() {
				_, err := kv.Get(ctx, "test", "key2")

				Expect(err).To(MatchError(core.ErrKeyNotFound))
			})
		})
	})

	Describe("Put", func() {
		Context("when key exists", func() {
			It("updates the value", func() {
				err := kv.Put(ctx, "test", "key", []byte("new - value"))

				Expect(err).ToNot(HaveOccurred())
				value, err := kv.Get(ctx, "test", "key")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("new - value")))
			})
		})

		Context("when key does not exists", func() {
			It("creates the value", func() {
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
			It("returns an error", func() {
				err := kv.Create(ctx, "test", "key", []byte("new - value"))

				Expect(err).To(MatchError(core.ErrKeyExists))
			})
		})

		Context("when key does not exists", func() {
			It("creates the value", func() {
				err := kv.Create(ctx, "test", "key2", []byte("new - value"))

				Expect(err).ToNot(HaveOccurred())
				value, err := kv.Get(ctx, "test", "key2")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("new - value")))
			})
		})
	})

	Describe("Delete", func() {
		Context("when key exists", func() {
			It("deletes the key", func() {
				err := kv.Delete(ctx, "test", "key")

				Expect(err).ToNot(HaveOccurred())
				_, err = kv.Get(ctx, "test", "key")

				Expect(err).To(MatchError(core.ErrKeyNotFound))
			})
		})

		Context("when key does not exists", func() {
			It("does not return an error", func() {
				err := kv.Delete(ctx, "test", "key2")

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("WaitCreated", func() {
		Context("when key exists", func() {
			It("returns the value", func() {
				value, err := kv.WaitCreated(ctx, "test", "key")

				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal([]byte("some - value")))
			})
		})

		Context("when key does not exists", func() {
			It("waits for the key to be created", func() {
				done := lo.Async0(func() {
					value, err := kv.WaitCreated(ctx, "test", "key3")

					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(Equal([]byte("new - value")))
				})
				time.Sleep(10 * time.Millisecond)

				lo.Must0(kv.Create(ctx, "test", "key3", []byte("new - value")))

				<-done
			})
		})
	})
})
