## toolbox - start
## Generated with https://github.com/bakito/toolbox

## Current working directory
TB_LOCALDIR ?= $(shell which cygpath > /dev/null 2>&1 && cygpath -m $$(pwd) || pwd)
## Location to install dependencies to
TB_LOCALBIN ?= $(TB_LOCALDIR)/bin
$(TB_LOCALBIN):
	if [ ! -e $(TB_LOCALBIN) ]; then mkdir -p $(TB_LOCALBIN); fi

# Helper functions
STRIP_V = $(patsubst v%,%,$(1))

## Tool Binaries
TB_GO_LICENSES ?= $(TB_LOCALBIN)/go-licenses
TB_GOLANGCI_LINT ?= $(TB_LOCALBIN)/golangci-lint
TB_GORELEASER ?= $(TB_LOCALBIN)/goreleaser
TB_SEMVER ?= $(TB_LOCALBIN)/semver

## Tool Versions
# renovate: packageName=github.com/google/go-licenses/v2
TB_GO_LICENSES_VERSION ?= v2.0.1
# renovate: packageName=github.com/golangci/golangci-lint/v2
TB_GOLANGCI_LINT_VERSION ?= v2.13.2
TB_GOLANGCI_LINT_VERSION_NUM ?= $(call STRIP_V,$(TB_GOLANGCI_LINT_VERSION))
# renovate: packageName=github.com/goreleaser/goreleaser/v2
TB_GORELEASER_VERSION ?= v2.18.1
# renovate: packageName=github.com/bakito/semver
TB_SEMVER_VERSION ?= v1.1.10
TB_SEMVER_VERSION_NUM ?= $(call STRIP_V,$(TB_SEMVER_VERSION))

## Tool Installer
.PHONY: tb.go-licenses
tb.go-licenses: ## Download go-licenses locally if necessary.
	@test -s $(TB_GO_LICENSES) || \
		GOBIN=$(TB_LOCALBIN) go install github.com/google/go-licenses/v2@$(TB_GO_LICENSES_VERSION)
.PHONY: tb.golangci-lint
tb.golangci-lint: ## Download golangci-lint locally if necessary.
	@test -s $(TB_GOLANGCI_LINT) && $(TB_GOLANGCI_LINT) --version | grep -q $(TB_GOLANGCI_LINT_VERSION_NUM) || \
		GOBIN=$(TB_LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(TB_GOLANGCI_LINT_VERSION)
.PHONY: tb.goreleaser
tb.goreleaser: ## Download goreleaser locally if necessary.
	@test -s $(TB_GORELEASER) || \
		GOBIN=$(TB_LOCALBIN) go install github.com/goreleaser/goreleaser/v2@$(TB_GORELEASER_VERSION)
.PHONY: tb.semver
tb.semver: ## Download semver locally if necessary.
	@test -s $(TB_SEMVER) && $(TB_SEMVER) -version | grep -q $(TB_SEMVER_VERSION_NUM) || \
		GOBIN=$(TB_LOCALBIN) go install github.com/bakito/semver@$(TB_SEMVER_VERSION)

## Reset Tools
.PHONY: tb.reset
tb.reset:
	@rm -f \
		$(TB_GO_LICENSES) \
		$(TB_GOLANGCI_LINT) \
		$(TB_GORELEASER) \
		$(TB_SEMVER)

## Update Tools
.PHONY: tb.update
tb.update: tb.reset
	toolbox makefile --renovate -f $(TB_LOCALDIR)/Makefile \
		github.com/google/go-licenses/v2 \
		github.com/golangci/golangci-lint/v2/cmd/golangci-lint?--version \
		github.com/goreleaser/goreleaser/v2 \
		github.com/bakito/semver?-version
## toolbox - end
