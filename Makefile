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

run:
	go run $(CMD_PATH)

doctor:
	go run $(CMD_PATH) doctor

test:
	go test ./... -v -race -cover

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

