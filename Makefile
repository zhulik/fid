check: lint test

lint_fix:
	 go tool github.com/golangci/golangci-lint/cmd/golangci-lint run --fix

lint:
	 go tool github.com/golangci/golangci-lint/cmd/golangci-lint run

test:
	go tool github.com/onsi/ginkgo/v2/ginkgo run -r -race

.PHONY: lint lint_fix test
