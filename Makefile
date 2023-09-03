SHELL=/bin/bash -o pipefail

.PHONY: install
install:
	go install ./cmd/candy/...

.PHONY: build
build:
	go build -o build/candy ./cmd/candy

.PHONY: vet
vet:
	docker run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:latest golangci-lint run -v --timeout 180

.PHONY: test
test:
	go test ./... -timeout=180s -coverprofile=c.out -covermode=atomic -count=1 -race -v
