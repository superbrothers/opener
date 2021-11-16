NAME := opener
DIST_DIR := dist
GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty)

PLATFORM ?= local
DOCKER ?= DOCKER_BUILDKIT=1 docker

.PHONY: build
build:
	$(DOCKER) build --target bin --output $(DIST_DIR) --platform $(PLATFORM) .

TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
GORELEASER_BIN := bin/goreleaser
GORELEASER := $(TOOLS_DIR)/$(GORELEASER_BIN)

$(GORELEASER): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR) && $(GO) build -o $(GORELEASER_BIN) github.com/goreleaser/goreleaser

.PHONY: build-cross
build-cross: $(GORELEASER)
	$(GORELEASER) build --snapshot --rm-dist

.PHONY: test
test:
	$(DOCKER) build --target test .

.PHONY: lint
lint:
	$(DOCKER) build --target lint .

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
