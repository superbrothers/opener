NAME := opener
DIST_DIR := dist
GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty)

PLATFORM ?= local
DOCKER ?= DOCKER_BUILDKIT=1 docker

.PHONY: build
build:
	$(DOCKER) build --target bin --output $(DIST_DIR) --platform $(PLATFORM) .

TOOLS_BIN_DIR := $(CURDIR)/hack/tools/bin
$(shell mkdir -p $(TOOLS_BIN_DIR))

# renovate: datasource=github-releases depName=goreleaser/goreleaser
GORELEASER_VERSION ?= v1.10.2
GORELEASER := $(TOOLS_BIN_DIR)/goreleaser

$(GORELEASER):
	GOBIN=$(TOOLS_BIN_DIR) $(GO) install github.com/goreleaser/goreleaser@$(GORELEASER_VERSION)

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
