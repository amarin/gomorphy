# see example at https://github.com/akiyosi/goneovim/blob/master/Makefile

#SHELL:=/bin/bash
#export SHELL

TAG := $(shell git describe --tags --abbrev=0)
VERSION := $(shell git describe --tags)
VERSION_HASH := $(shell git rev-parse HEAD)
PRODUCER := com.amarin.gomorphy

ORIGIN:=imgdesk
ORIGIN_MAC:=$(ORIGIN).app
PACKAGES=pkg

# deployment directory
DEPLOYMENT_PREFIX:=./deploy

# Go parameters
GOCMD=GO111MODULE=on go
DEPCMD=$(GOCMD) mod
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Detect GOOS. By default build release for current OS. Redefine GOOS to cross-build.
GOOS ?= $(shell go env GOOS)

.PHONY: clean debug run build build-docker-linux build-docker-windows

# If the first argument is "run"...
ifeq (debug,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "run"
  DEBUG_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(DEBUG_ARGS):;@:)
endif

all: build

lint: ## Lint the files
	@golangci-lint run

tidy: ## Fix dependencies records in go.mod
	@go mod tidy

deps: tidy ## Get the dependencies
	$(DEPCMD) vendor

race: deps ## Run data race detector
	@go test -race -short ${PKG_LIST}

msan: deps ## Run memory sanitizer
	@go test -msan -short ${PKG_LIST}

make_deploy:
	@echo "make deployment at ./$(DEPLOYMENT_PREFIX)"
	@mkdir -p ./$(DEPLOYMENT_PREFIX)

opencorpora_update: make_deploy
	@echo "build $@ at $(DEPLOYMENT_PREFIX)"
	${GOBUILD} -o $(DEPLOYMENT_PREFIX)/opencorpora_update ./cmd/opencorpora_update/main.go

opencorpora_test: make_deploy
	@echo "build $@ at $(DEPLOYMENT_PREFIX)"
	${GOBUILD} -o $(DEPLOYMENT_PREFIX)/opencorpora_test ./cmd/opencorpora_test/main.go

build: opencorpora_update ## build executable for target os

debug:
	@export GO111MODULE=off
	cd $(CMDMAINPATH)
	test -f ../../$(PACKAGES)/moc.go & $(CMD_QT_MOC)
	dlv debug --build-flags -race -- $(DEBUG_ARGS)

run:
	@export GO111MODULE=off
	cd $(CMDMAINPATH)
	test -f ../../$(PACKAGES)/moc.go & $(CMD_QT_MOC)
	go run $(CMDMAINPATH)/main.go

clean:
	@export GO111MODULE=off
	rm -fr $(CMDMAINPATH)/deploy/*
	rm -fr $(PACKAGES)/*moc*

## Display this help screen
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
