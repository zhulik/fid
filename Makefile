all:
	$(MAKE) -j check
	$(MAKE) -j build

check: lint test

COMPONENTS := gateway runtimeapi infoserver scaler demo-function

lint_fix:
	 go tool github.com/golangci/golangci-lint/cmd/golangci-lint run --fix

lint:
	 go tool github.com/golangci/golangci-lint/cmd/golangci-lint run

test:
	go tool github.com/onsi/ginkgo/v2/ginkgo run -r -race

build:
	$(MAKE) -j $(COMPONENTS)

$(COMPONENTS):
	docker buildx build --build-arg COMPONENT=$@ -t ghcr.io/zhulik/fid-$@ .


.PHONY: lint lint_fix test build
