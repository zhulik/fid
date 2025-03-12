package docker_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/zhulik/fid/internal/backends/docker"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
	"github.com/zhulik/fid/testhelpers"
)

const (
	instanceID   = "some-ID"
	instanceID1  = "some-ID1"
	functionName = "some-function"
)

var function = docker.Function{
	Name_: functionName,
}

var _ = Describe("InstancesRepo", Serial, func() {
	var injector *do.Injector
	var repo *docker.InstancesRepo
	var kv core.KV

	BeforeEach(func(ctx SpecContext) {
		injector = testhelpers.NewInjector()
		kv = lo.Must(nats.NewKV(injector))

		lo.Must(kv.CreateBucket(ctx, core.BucketNameInstances, 0))

		DeferCleanup(func(ctx SpecContext) { kv.DeleteBucket(ctx, core.BucketNameInstances) }) //nolint:errcheck

		do.Provide(injector, func(injector *do.Injector) (core.KV, error) {
			return kv, nil
		})

		repo = lo.Must(docker.NewInstancesRepo(ctx, injector))
	})

	Describe("Add", func() {
		Context("when instance does not exist", func() {
			It("creates a new instance", func(ctx SpecContext) {
				err := repo.Add(ctx, function, instanceID)
				Expect(err).ToNot(HaveOccurred())

				instance, err := repo.Get(ctx, function, instanceID)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.ID()).To(Equal(instanceID))
			})

			Context("when instance already exists", func() {
				BeforeEach(func(ctx SpecContext) {
					lo.Must0(repo.Add(ctx, function, instanceID))
				})

				It("returns an error", func(ctx SpecContext) {
					err := repo.Add(ctx, function, instanceID)
					Expect(err).To(MatchError(core.ErrInstanceAlreadyExists))
				})
			})
		})
	})

	Describe("SetLastExecuted", func() {
		Describe("SetLastExecuted", func() {
			lastExecuted := time.Now()

			BeforeEach(func(ctx SpecContext) {
				lo.Must0(repo.Add(ctx, function, instanceID))
			})

			It("updates the LastExecuted timestamp", func(ctx SpecContext) {
				err := repo.SetLastExecuted(ctx, function, instanceID, lastExecuted)
				Expect(err).ToNot(HaveOccurred())

				instance, err := repo.Get(ctx, function, instanceID)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.LastExecuted()).To(BeTemporally("~", lastExecuted, time.Second))
			})
		})

		Describe("SetBusy", func() {
			BeforeEach(func(ctx SpecContext) {
				lo.Must0(repo.Add(ctx, function, instanceID))
			})

			It("updates the busy status", func(ctx SpecContext) {
				err := repo.SetBusy(ctx, function, instanceID, true)
				Expect(err).ToNot(HaveOccurred())

				instance, err := repo.Get(ctx, function, instanceID)
				Expect(err).ToNot(HaveOccurred())
				Expect(instance.Busy()).To(BeTrue())
			})
		})

		Describe("CountIdle", func() {
			Context("when no instances exist", func() {
				It("returns 0", func(ctx SpecContext) {
					idle, err := repo.CountIdle(ctx, function)
					Expect(err).ToNot(HaveOccurred())
					Expect(idle).To(BeZero())
				})
			})

			Context("when instances exist", func() {
				BeforeEach(func(ctx SpecContext) {
					lo.Must0(repo.Add(ctx, function, instanceID))
					lo.Must0(repo.Add(ctx, function, instanceID1))
				})

				Context("when all instances are busy", func() {
					BeforeEach(func(ctx SpecContext) {
						lo.Must0(repo.SetBusy(ctx, function, instanceID, true))
						lo.Must0(repo.SetBusy(ctx, function, instanceID1, true))
					})

					It("returns the number of idle instances", func(ctx SpecContext) {
						idle, err := repo.CountIdle(ctx, function)
						Expect(err).ToNot(HaveOccurred())
						Expect(idle).To(BeZero())
					})
				})

				Context("when some instances are busy", func() {
					BeforeEach(func(ctx SpecContext) {
						lo.Must0(repo.SetBusy(ctx, function, instanceID, true))
					})

					It("returns the number of idle instances", func(ctx SpecContext) {
						idle, err := repo.CountIdle(ctx, function)
						Expect(err).ToNot(HaveOccurred())
						Expect(idle).To(Equal(1))
					})
				})
			})

			It("updates the busy status", func(ctx SpecContext) {
				err := repo.SetBusy(ctx, function, instanceID, true)
				Expect(err).ToNot(HaveOccurred())

				idle, err := repo.CountIdle(ctx, function)
				Expect(err).ToNot(HaveOccurred())
				Expect(idle).To(BeZero())
			})
		})
	})

	Describe("Get", func() {
		Context("when instance exists", func() {
			BeforeEach(func(ctx SpecContext) {
				lo.Must0(repo.Add(ctx, function, instanceID))
			})

			It("returns the instance", func(ctx SpecContext) {
				instance, err := repo.Get(ctx, function, instanceID)

				Expect(err).ToNot(HaveOccurred())
				Expect(instance.ID()).To(Equal(instanceID))
				Expect(instance.Function()).To(Equal(function))
				Expect(instance.LastExecuted()).To(Equal(time.Time{}))
			})
		})

		Context("when instance does not exist", func() {
			It("returns an error", func(ctx SpecContext) {
				_, err := repo.Get(ctx, function, instanceID)

				Expect(err).To(MatchError(core.ErrInstanceNotFound))
			})
		})
	})

	Describe("List", func() {
		Context("when no instances exist", func() {
			It("returns an empty list", func(ctx SpecContext) {
				instances, err := repo.List(ctx, function)

				Expect(err).ToNot(HaveOccurred())
				Expect(instances).To(BeEmpty())
			})
		})

		Context("when instances exist", func() {
			lastExecuted := time.Now()

			BeforeEach(func(ctx SpecContext) {
				lo.Must0(repo.Add(ctx, function, instanceID))
				lo.Must0(repo.Add(ctx, function, instanceID1))

				lo.Must0(repo.SetBusy(ctx, function, instanceID1, true))
				lo.Must0(repo.SetLastExecuted(ctx, function, instanceID1, lastExecuted))
			})

			It("returns instances", func(ctx SpecContext) {
				instances, err := repo.List(ctx, function)

				Expect(err).ToNot(HaveOccurred())
				Expect(instances).To(HaveLen(2))
				Expect(instances[0].ID()).To(Equal(instanceID))
				Expect(instances[0].Function()).To(Equal(function))
				Expect(instances[0].Busy()).To(BeFalse())

				Expect(instances[0].LastExecuted()).To(Equal(time.Time{}))

				Expect(instances[1].ID()).To(Equal(instanceID1))
				Expect(instances[1].Function()).To(Equal(function))
				Expect(instances[1].Busy()).To(BeTrue())

				Expect(instances[1].LastExecuted()).To(BeTemporally("~", lastExecuted, 10*time.Millisecond))
			})
		})
	})

	Describe("Count", func() {
		Context("when no instances exist", func() {
			It("returns 0", func(ctx SpecContext) {
				count, err := repo.Count(ctx, function)

				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(BeZero())
			})
		})

		Context("when instances exist", func() {
			BeforeEach(func(ctx SpecContext) {
				lo.Must0(repo.Add(ctx, function, instanceID))
			})

			It("returns instances", func(ctx SpecContext) {
				count, err := repo.Count(ctx, function)

				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(1))
			})
		})
	})

	Describe("Delete", func() {
		Context("when instance does not exist", func() {
			It("returns an error", func(ctx SpecContext) {
				err := repo.Delete(ctx, function, instanceID)

				Expect(err).To(MatchError(core.ErrInstanceNotFound))
			})
		})

		Context("when instance exists", func() {
			BeforeEach(func(ctx SpecContext) {
				lo.Must0(repo.Add(ctx, function, instanceID))
			})

			It("deletes the instance", func(ctx SpecContext) {
				err := repo.Delete(ctx, function, instanceID)

				Expect(err).ToNot(HaveOccurred())

				_, err = repo.Get(ctx, function, instanceID)

				Expect(err).To(MatchError(core.ErrInstanceNotFound))
			})
		})
	})
})
