package docker_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/backends/docker"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/kv/nats"
	natsPubSub "github.com/zhulik/fid/internal/pubsub/nats"
)

var _ = Describe("InstancesRepo", func() {
	var injector *do.Injector
	var repo core.InstancesRepo

	BeforeEach(func() {
		injector = do.New()
		do.ProvideValue[logrus.FieldLogger](injector, logrus.New())

		do.ProvideValue[core.Config](injector, config.Config{})
		do.Provide(injector, natsPubSub.NewClient)

		do.Provide[core.KV](injector, func(injector *do.Injector) (core.KV, error) {
			return nats.NewKV(injector)
		})

		do.Provide[core.FunctionsRepo](injector, func(injector *do.Injector) (core.FunctionsRepo, error) {
			return docker.NewFunctionsRepo(injector)
		})

		repo = lo.Must(docker.NewInstancesRepo(injector))
	})

	Describe("Upsert", func() {
		It("creates a new instance", func(ctx SpecContext) {
			err := repo.Upsert(ctx, docker.FunctionInstance{
				ID_:           "some-ID",
				LastExecuted_: time.Now(),
				Function_: docker.Function{
					Name_: "some-function",
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Get", func() {
	})

	Describe("List", func() {
	})

	Describe("Delete", func() {
	})
})
