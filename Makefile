GOLANGCI_LINT_VERSION = "1.63.4"

check: lint test

lint_fix: | bin/golangci-lint
	./bin/golangci-lint run --fix

lint: | bin/golangci-lint
	./bin/golangci-lint run

test:
	go run github.com/onsi/ginkgo/v2/ginkgo -r

bin/golangci-lint:
	set -eu
	curl --silent \
		 --fail \
		 --location \
         https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s v$(GOLANGCI_LINT_VERSION)


.PHONY: lint lint_fix test
