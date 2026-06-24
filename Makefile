# ---- Config ----

APP_NAME := relayops
CMD_PATH := ./cmd/relayops
BUILD_DIR := ./bin

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "\
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildDate=$(DATE)"

# ---- Targets ----

.PHONY: all build run test clean install fmt vet tidy doctor

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_PATH)
	@echo "Built $(BUILD_DIR)/$(APP_NAME)"

agent:
	go build -o ./bin/relayops-agent ./cmd/relayops-agent

run:
	go run $(CMD_PATH)

doctor:
	go run $(CMD_PATH) doctor

test:
	go test ./... -v -race -coverpkg=./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -n 20

cover:
	go test ./... -coverpkg=./... -coverprofile=coverage.out
	gocover-cobertura < coverage.out > coverage.xml
	go tool cover -func=coverage.out

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BUILD_DIR)
	@echo "Cleaned build artifacts"

install:
	go install $(LDFLAGS) $(CMD_PATH)

BIN := $(HOME)/bin

install-scripts:
	mkdir -p $(BIN)

	ln -sf $(PWD)/scripts/ts890/start-ardop  $(BIN)/start-ardop-ts890
	ln -sf $(PWD)/scripts/ts890/stop-ardop   $(BIN)/stop-ardop-ts890

	ln -sf $(PWD)/scripts/ts890/start-rigctl $(BIN)/start-rigctl-ts890
	ln -sf $(PWD)/scripts/ts890/stop-rigctl  $(BIN)/stop-rigctl-ts890

	ln -sf $(PWD)/scripts/ic9700/start-packet $(BIN)/start-packet-ic9700
	ln -sf $(PWD)/scripts/ic9700/stop-packet  $(BIN)/stop-packet-ic9700

	ln -sf $(PWD)/scripts/ic9700/start-rigctl $(BIN)/start-rigctl-ic9700
	ln -sf $(PWD)/scripts/ic9700/stop-rigctl  $(BIN)/stop-rigctl-ic9700