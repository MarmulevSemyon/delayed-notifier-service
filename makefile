
APP_NAME := delayedNotifier
BIN_DIR  := bin
CMD_DIR  := ./cmd/delayedNotifier

GO      := go
GOFMT   := gofmt
GOLINT  := golint

.PHONY: all fmt vet lint test build run clean

all: fmt vet lint test build

fmt:
	$(GOFMT) -w ./cmd ./internal 

vet:
	$(GO) vet ./...

lint:
	$(GOLINT) ./...

test: build
	$(GO) test ./...

race: fmt vet lint build
	$(GO) run -race $(CMD_DIR)
build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-logs-delay:
	docker compose logs -f delayed_notifier

clean:
	rm -rf $(BIN_DIR)