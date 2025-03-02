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
	"github.com/zhulik/fid/testhelpers/mocks"
)

const (
	instanceID   = "some-ID"
	functionName = "some-function"
)

var (
	function = docker.Function{
		Name_: functionName,
	}
	functionInstance = docker.FunctionInstance{
		ID_:           instanceID,
		LastExecuted_: time.Now(),
		Function_:     function,
	}
)

var _ = Describe("InstancesRepo", Serial, func() {
	var injector *do.Injector
	var repo *docker.InstancesRepo
	var functionsRepoMock *mocks.MockFunctionsRepo
	var kv core.KV

	BeforeEach(func(ctx SpecContext) {
		functionsRepoMock = mocks.NewMockFunctionsRepo(GinkgoT())

		injector = testhelpers.NewInjector()
		kv = lo.Must(nats.NewKV(injector))

		lo.Must(kv.CreateBucket(ctx, core.BucketNameInstances, 0))

		DeferCleanup(func(ctx SpecContext) { kv.DeleteBucket(ctx, core.BucketNameInstances) }) //nolint:errcheck

		do.Provide(injector, func(injector *do.Injector) (core.KV, error) {
			return kv, nil
		})
		do.Provide(injector, func(injector *do.Injector) (core.FunctionsRepo, error) {
			return functionsRepoMock, nil
		})

		repo = lo.Must(docker.NewInstancesRepo(injector))
	})

	Describe("Upsert", func() {
		It("creates a new instance", func(ctx SpecContext) {
			err := repo.Upsert(ctx, functionInstance)

			Expect(err).ToNot(HaveOccurred())

			functionsRepoMock.On("Get", ctx, functionName).Return(function, nil).Once()

			_, err = repo.Get(ctx, instanceID)

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Get", func() {
		Context("when instance exists", func() {})

		Context("when instance exists", func() {
			BeforeEach(func(ctx SpecContext) {
				lo.Must0(repo.Upsert(ctx, functionInstance))
			})

			Context("when function does not exist", func() {
				It("returns an error", func(ctx SpecContext) {
					functionsRepoMock.On("Get", ctx, functionName).Return(nil, core.ErrFunctionNotFound).Once()

					_, err := repo.Get(ctx, instanceID)

					Expect(err).To(MatchError(core.ErrFunctionNotFound))
				})
			})

			Context("when function exists", func() {
				It("returns the instance", func(ctx SpecContext) {
					functionsRepoMock.On("Get", ctx, functionName).Return(function, nil).Once()

					instance, err := repo.Get(ctx, instanceID)

					Expect(err).ToNot(HaveOccurred())
					Expect(instance.ID()).To(Equal(instanceID))
					Expect(instance.Function()).To(Equal(function))
					Expect(instance.LastExecuted().UnixNano()).To(Equal(functionInstance.LastExecuted().UnixNano()))
				})
			})
		})

		Context("when instance does not exist", func() {
			It("returns an error", func(ctx SpecContext) {
				_, err := repo.Get(ctx, instanceID)

				Expect(err).To(MatchError(core.ErrInstanceNotFound))
			})
		})
	})

	Describe("List", func() {
		Context("when function does not exist", func() {
			It("returns an error", func(ctx SpecContext) {
				functionsRepoMock.On("Get", ctx, functionName).Return(nil, core.ErrFunctionNotFound).Once()

				_, err := repo.List(ctx, functionName)

				Expect(err).To(MatchError(core.ErrFunctionNotFound))
			})
		})

		Context("when function exists", func() {
			Context("when no instances exist", func() {
				It("returns an empty list", func(ctx SpecContext) {
					functionsRepoMock.On("Get", ctx, functionName).Return(function, nil).Once()

					instances, err := repo.List(ctx, functionName)

					Expect(err).ToNot(HaveOccurred())
					Expect(instances).To(BeEmpty())
				})
			})

			Context("when instances exist", func() {
				BeforeEach(func(ctx SpecContext) {
					lo.Must0(repo.Upsert(ctx, functionInstance))
				})

				It("returns instances", func(ctx SpecContext) {
					functionsRepoMock.On("Get", ctx, functionName).Return(function, nil).Once()

					instances, err := repo.List(ctx, functionName)

					Expect(err).ToNot(HaveOccurred())
					Expect(instances).To(HaveLen(1))
					Expect(instances[0].ID()).To(Equal(instanceID))
					Expect(instances[0].Function()).To(Equal(function))
					Expect(instances[0].LastExecuted().UnixNano()).To(Equal(functionInstance.LastExecuted().UnixNano()))
				})
			})
		})
	})

	Describe("Delete", func() {
		Context("when instance does not exist", func() {
			It("returns an error", func(ctx SpecContext) {
				err := repo.Delete(ctx, instanceID)

				Expect(err).To(MatchError(core.ErrInstanceNotFound))
			})
		})

		Context("when instance exists", func() {
			BeforeEach(func(ctx SpecContext) {
				lo.Must0(repo.Upsert(ctx, functionInstance))
			})

			Context("when function does not exist", func() {
				It("returns an error", func(ctx SpecContext) {
					functionsRepoMock.On("Get", ctx, functionName).Return(nil, core.ErrFunctionNotFound).Once()

					err := repo.Delete(ctx, instanceID)

					Expect(err).To(MatchError(core.ErrFunctionNotFound))
				})
			})

			Context("when function exists", func() {
				It("deletes the instance", func(ctx SpecContext) {
					functionsRepoMock.On("Get", ctx, functionName).Return(function, nil).Once()

					err := repo.Delete(ctx, instanceID)

					Expect(err).ToNot(HaveOccurred())

					_, err = repo.Get(ctx, instanceID)

					Expect(err).To(MatchError(core.ErrInstanceNotFound))
				})
			})
		})
	})
})
