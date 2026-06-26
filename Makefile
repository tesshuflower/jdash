BINARY_NAME := jdash

# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.12.2
GOLANGCI_LINT := ./bin/golangci-lint

.PHONY: build test lint run clean

build:
	go build -o $(BINARY_NAME) .

test:
	go test ./...

$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b ./bin $(GOLANGCI_LINT_VERSION)

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

run:
	go run .

clean:
	rm -f $(BINARY_NAME)
	rm -rf ./bin
