package nats_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
	"github.com/zhulik/fid/testhelpers"
)

var _ = Describe("Nats KV", Serial, func() {
	var injector *do.Injector
	var kv core.KV

	BeforeEach(func(ctx SpecContext) {
		injector = testhelpers.NewInjector()

		kv = lo.Must(nats.NewKV(injector))

		lo.Must(kv.CreateBucket(ctx, "test", 0))
		DeferCleanup(func(ctx SpecContext) { kv.DeleteBucket(ctx, "test") }) //nolint:errcheck
	})

	Describe("CreateBucket", func() {
		Context("when bucket exists", func() {
			It("does not return an error", func(ctx SpecContext) {
				_, err := kv.CreateBucket(ctx, "test", 0)

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when bucket does not exists", func() {
			It("creates the bucket", func(ctx SpecContext) {
				_, err := kv.CreateBucket(ctx, "test2", 0)
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

	Describe("Bucket", func() {
		Context("when bucket exists", func() {
			It("returns a bucket", func(ctx SpecContext) {
				bucket, err := kv.Bucket(ctx, "test")

				Expect(err).ToNot(HaveOccurred())
				Expect(bucket).ToNot(BeNil())
			})
		})

		Context("when bucket does not exists", func() {
			It("returns an error", func(ctx SpecContext) {
				_, err := kv.Bucket(ctx, "test2")

				Expect(err).To(MatchError(core.ErrBucketNotFound))
			})
		})
	})
})
