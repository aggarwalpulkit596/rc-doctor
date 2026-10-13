.PHONY: build test check fmt fmt-check vet lint cover golden snapshot install-hooks

build:
	go build -o rc-doctor ./cmd/rc-doctor

test:
	go test -race ./...

cover:
	go test -coverpkg=./... -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

fmt:
	gofmt -w .

fmt-check:
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then \
		echo "unformatted files:"; echo "$$out"; exit 1; \
	fi

vet:
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not installed: see mise.toml, or go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2"; exit 1; }
	golangci-lint run ./...

# What CI runs, minus govulncheck. Lint is skipped with a note when
# golangci-lint isn't installed; CI still enforces it.
check: fmt-check vet test
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed: skipping lint locally; CI enforces it"; \
	fi

# Rewrite testdata/golden after an intended output change, then review the diff.
golden:
	go test ./cmd/rc-doctor -run TestGolden -update

# Build every release archive locally, without publishing.
snapshot:
	goreleaser release --snapshot --clean

install-hooks:
	@command -v lefthook >/dev/null || go install github.com/evilmartians/lefthook@latest
	@if command -v lefthook >/dev/null; then \
		lefthook install; \
	else \
		PATH="$(shell go env GOPATH)/bin:$$PATH" lefthook install; \
	fi
