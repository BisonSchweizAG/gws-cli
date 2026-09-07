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

release: tb.goreleaser tb.semver licenses
	@version=$$($(TB_SEMVER)); \
	git tag -s $$version -m"Release $$version"; \
	git push origin $$version
	$(TB_GORELEASER) --clean

test-release: tb.goreleaser licenses
	GOOGLE_OIDC_CLIENT_ID=test-client GOOGLE_OIDC_CLIENT_SECRET=test-secret \
	  $(TB_GORELEASER) --skip=publish --snapshot --clean

release-ci: tb.goreleaser licenses
	$(TB_GORELEASER) --clean

fmt: tb.golines tb.gofumpt
	$(TB_GOLINES) --base-formatter="$(TB_GOFUMPT)" --max-len=120 --write-output .

build-win:
	GOOS=windows GOARCH=amd64 go build -o gws.exe -ldflags="-s -w -X github.com/bisonschweizag/gws-cli/version.Version=dev-$$(date +%Y%m%d-%H%M)" .

check-vulnerabilities:
	go run golang.org/x/vuln/cmd/govulncheck@latest -show verbose,color ./...

# Extract licenses
licenses: tb.go-licenses
	rm -rf licenses licenses.csv licenses.md
	$(TB_GO_LICENSES) save ./... --save_path=licenses --force 2>/dev/null
	$(TB_GO_LICENSES) report ./... > licenses.csv 2>/dev/null
	@awk -F, 'BEGIN { print "# Licenses\n\n| Module | License | Type |"; print "| --- | --- | --- |" } { print "| `" $$1 "` | [License](" $$2 ") | " $$3 " |" }' licenses.csv > licenses.md