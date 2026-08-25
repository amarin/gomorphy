GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test

DEPLOYMENT_PREFIX=./deploy

.PHONY: all build lint tidy deps race test update compile clean help

all: build

build: ## Build CLI binaries
	@mkdir -p $(DEPLOYMENT_PREFIX)
	$(GOBUILD) -o $(DEPLOYMENT_PREFIX)/gomorphy ./cmd/gomorphy
	$(GOBUILD) -o $(DEPLOYMENT_PREFIX)/opencorpora_update ./cmd/opencorpora_update

update: ## Download and compile OpenCorpora dictionary
	$(DEPLOYMENT_PREFIX)/opencorpora_update -l

compile: ## Compile existing dict.xml to .dat
	$(DEPLOYMENT_PREFIX)/opencorpora_update -l -v

test: ## Run all unit tests with race detector
	$(GOTEST) -race -count=1 ./...

test-integration: ## Run integration tests (requires compiled dictionary)
	$(GOTEST) -race -tags integration -count=1 -run 'Integration|FullDict' ./pkg/dictionary/

lint: ## Run golangci-lint
	@golangci-lint run

tidy: ## Tidy go.mod
	@go mod tidy

deps: tidy ## Vendor dependencies
	@go mod vendor

clean: ## Remove build artifacts
	rm -rf $(DEPLOYMENT_PREFIX)

help: ## Display this help
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
