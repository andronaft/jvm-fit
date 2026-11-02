BIN := bin/jvm-fit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test vet lint evidence evidence-table
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BIN) ./cmd/jvm-fit

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

# Compare the model with real JVMs in Docker (needs Docker).
evidence:
	go test -tags evidence -run TestDockerEvidence -v ./internal/model/

# Regenerate docs/evidence/results.md (needs Docker, takes a few minutes).
evidence-table:
	docs/evidence/run.sh > docs/evidence/results.md
