SHELL=/bin/bash -o pipefail

.PHONY: build
build:
	go build -o build/candy -mod=vendor ./cmd/candy

.PHONY: vet
vet:
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v

.PHONY: test
test:
	go test ./... -timeout=120s -coverprofile=c.out -covermode=atomic -mod=vendor -count=1 -race -v
