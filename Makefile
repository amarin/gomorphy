GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test

DEPLOYMENT_PREFIX=./deploy

.PHONY: all build lint tidy deps race test update compile clean help

all: build
CLI_MAIN=gomorphy

$(DEPLOYMENT_PREFIX):
	@echo "make folder for compiled binaries at ${DEPLOYMENT_PREFIX}"
	@mkdir -p $(DEPLOYMENT_PREFIX)

$(DEPLOYMENT_PREFIX)/${CLI_MAIN}: $(DEPLOYMENT_PREFIX)
	@echo "build ${CLI_MAIN}"
	$(GOBUILD) -o $(DEPLOYMENT_PREFIX)/${CLI_MAIN} ./cmd/gomorphy

build: $(DEPLOYMENT_PREFIX)/${CLI_MAIN} ## Build CLI binary

update: $(DEPLOYMENT_PREFIX)/${CLI_MAIN} ## Download, unpack and build OpenCorpora dictionary
	$(DEPLOYMENT_PREFIX)/${CLI_MAIN} update opencorpora

compile: $(DEPLOYMENT_PREFIX)/${CLI_MAIN} ## Compile existing dict.xml to .dat
	$(DEPLOYMENT_PREFIX)/${CLI_MAIN} build opencorpora

test: ## Run all unit tests with race detector
	$(GOTEST) -race -count=1 ./...

test-integration: ## Run integration tests (network access + real OpenCorpora data required)
	$(GOTEST) -race -tags integration -count=1 ./...

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
