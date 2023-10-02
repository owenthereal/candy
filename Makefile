SHELL=/bin/bash -o pipefail

.PHONY: install
install:
	go install ./cmd/candy/...

.PHONY: build
build:
	go build -o bin/candy ./cmd/candy

.PHONY: vet
vet:
	docker run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:latest golangci-lint run -v

.PHONY: test
test:
	go test ./... -timeout=5m -coverprofile=c.out -covermode=atomic -count=1 -race -v

BIN_DIR ?= $(CURDIR)/bin
export PATH := $(BIN_DIR):$(PATH)
.PHONY: tools
tools:
	# goreleaser
	GOBIN=$(BIN_DIR) go install github.com/goreleaser/goreleaser@latest

.PHONY: goreleaser
goreleaser:
	goreleaser release --clean --snapshot --skip=publish
