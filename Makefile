SHELL := bash

VERSION          ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
TAILWIND_VERSION ?= v4.1.11
GOLANGCI_VERSION ?= v2.14.0
TEST_DATABASE_URL ?= postgres://app:app@localhost:5432/app?sslmode=disable

# Tailwind standalone CLI for this machine.
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_S),Darwin)
  TW_OS := macos
else ifneq (,$(findstring MINGW,$(UNAME_S)))
  TW_OS := windows
else
  TW_OS := linux
endif
ifneq (,$(filter arm64 aarch64,$(UNAME_M)))
  TW_ARCH := arm64
else
  TW_ARCH := x64
endif
TW_EXT := $(if $(filter windows,$(TW_OS)),.exe,)
TAILWIND := bin/tailwindcss-$(TAILWIND_VERSION)$(TW_EXT)

.PHONY: all generate css build run dev test test-db lint docker db-up db-down clean

all: generate css lint test build

generate: ## Regenerate templ code (committed; CI checks it is current)
	go tool templ generate

$(TAILWIND):
	@mkdir -p bin
	curl -fsSL -o $@ https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(TW_OS)-$(TW_ARCH)$(TW_EXT)
	chmod +x $@

css: $(TAILWIND)
	$(TAILWIND) -i assets/css/input.css -o assets/css/output.css --minify

build: generate css
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/server ./cmd/server

run: generate css
	go run ./cmd/server

dev: $(TAILWIND) ## Rebuild templ + restart on change; run `make css-watch` alongside
	go tool templ generate --watch --cmd "go run ./cmd/server"

css-watch: $(TAILWIND)
	$(TAILWIND) -i assets/css/input.css -o assets/css/output.css --watch

test:
	go test -race ./...

test-db: db-up ## Includes store tests against the compose Postgres
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -race ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION) run

docker:
	docker build --build-arg VERSION=$(VERSION) -t $(notdir $(CURDIR)):$(VERSION) .

db-up:
	docker compose up -d --wait db

db-down:
	docker compose down

clean:
	rm -rf bin assets/css/output.css
