# Include toolbox tasks
include ./.toolbox.mk

# Run go golanci-lint
lint: tb.golangci-lint
	$(TB_GOLANGCI_LINT) run --fix

# Run go mod tidy
tidy:
	go mod tidy

# Run tests
test:
	go test ./... -v -coverprofile=coverage.out

release: tb.goreleaser tb.semver tb.syft tb.goversioninfo
	@version=$$($(TB_SEMVER)); \
	git tag -s $$version -m"Release $$version"; \
	git push origin $$version
	PATH=$(TB_LOCALBIN):$${PATH} $(TB_GORELEASER) --clean

test-release: tb.goreleaser tb.syft tb.goversioninfo
	@rm -f resource_windows*.syso
	PATH=$(TB_LOCALBIN):$${PATH} GOOGLE_OIDC_CLIENT_ID=test-client GOOGLE_OIDC_CLIENT_SECRET=test-secret \
	  $(TB_GORELEASER) --skip=publish --snapshot --clean

release-ci: tb.goreleaser tb.syft tb.goversioninfo
	PATH=$(TB_LOCALBIN):$${PATH} $(TB_GORELEASER) --clean

fmt: tb.golines tb.gofumpt
	$(TB_GOLINES) --base-formatter="$(TB_GOFUMPT)" --max-len=120 --write-output .

build-win:
	GOOS=windows GOARCH=amd64 go build -trimpath -o gws.exe -ldflags="-s -w -X github.com/bisonschweizag/gws-cli/version.Version=dev-$$(date +%Y%m%d-%H%M)" .

check-vulnerabilities:
	go run golang.org/x/vuln/cmd/govulncheck@latest -show verbose,color ./...
