GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test

DEPLOYMENT_PREFIX=./deploy

.PHONY: all build lint tidy deps race test update compile clean help

all: build
CLI_MAIN=gomorphy
CLI_UPDATER=gomorphy_build

$(DEPLOYMENT_PREFIX):
	@echo "make folder for compiled binaries at ${DEPLOYMENT_PREFIX}"
	@mkdir -p $(DEPLOYMENT_PREFIX)

$(DEPLOYMENT_PREFIX)/${CLI_MAIN}: $(DEPLOYMENT_PREFIX)
	@echo "build ${CLI_MAIN}"
	$(GOBUILD) -o $(DEPLOYMENT_PREFIX)/${CLI_MAIN} ./cmd/gomorphy

$(DEPLOYMENT_PREFIX)/${CLI_UPDATER}: $(DEPLOYMENT_PREFIX)
	@echo "build ${CLI_UPDATER}"
	$(GOBUILD) -o $(DEPLOYMENT_PREFIX)/${CLI_UPDATER} ./cmd/gomorphy_build

build: $(DEPLOYMENT_PREFIX)/${CLI_MAIN} $(DEPLOYMENT_PREFIX)/${CLI_UPDATER} ## Build CLI binaries

update: $(DEPLOYMENT_PREFIX)/${CLI_UPDATER} ## Download, unpack and compile OpenCorpora dictionary
	$(DEPLOYMENT_PREFIX)/${CLI_UPDATER} update

compile: $(DEPLOYMENT_PREFIX)/${CLI_UPDATER} ## Compile existing dict.xml to .dat
	$(DEPLOYMENT_PREFIX)/${CLI_UPDATER} compile

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
