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

var _ = Describe("Nats KV", Serial, func() {
	var p *pal.Pal
	var kv nats.KV

	BeforeEach(func(ctx SpecContext) {
		p = testhelpers.NewPal(ctx)

		lo.Must0(pal.InjectInto(ctx, p, &kv))

		lo.Must(kv.CreateBucket(ctx, "test"))
		DeferCleanup(func(ctx SpecContext) { kv.DeleteBucket(ctx, "test") }) //nolint:errcheck
	})

	Describe("CreateBucket", func() {
		Context("when bucket exists", func() {
			It("does not return an error", func(ctx SpecContext) {
				_, err := kv.CreateBucket(ctx, "test")

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when bucket does not exists", func() {
			It("creates the bucket", func(ctx SpecContext) {
				_, err := kv.CreateBucket(ctx, "test2")
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
