NAME := opener
DIST_DIR := dist
GO ?= GOPROXY=https://gocenter.io go
VERSION ?= $(shell git describe --tags --always --dirty)

.PHONY: build
build:
	$(GO) build -o $(DIST_DIR)/$(NAME) .

TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
GORELEASER_BIN := bin/goreleaser
GORELEASER := $(TOOLS_DIR)/$(GORELEASER_BIN)

$(GORELEASER): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR) && $(GO) build -o $(GORELEASER_BIN) github.com/goreleaser/goreleaser

.PHONY: build-cross
build-cross: $(GORELEASER)
	$(GORELEASER) build --snapshot --rm-dist

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: lint
lint: vet fmt
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.31.0 golangci-lint run

.PHONY: dist
dist: $(GORELEASER)
	$(GORELEASER) release --rm-dist --skip-publish --snapshot

.PHONY: release
release: $(GORELEASER)
	$(GORELEASER) release --rm-dist --skip-validate

.PHONY: clean
clean: clean-tools clean-dist

.PHONY: clean-tools
clean-tools:
	rm -rf $(TOOLS_BIN_DIR)

.PHONY: clean-dist
clean-dist:
	rm -rf $(DIST_DIR)
