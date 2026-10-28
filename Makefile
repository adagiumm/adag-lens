BINARY  := adag-lens
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/adagiumm/adag-lens/internal/version.Version=$(VERSION)

.PHONY: all build test vet fmt fmt-check check clean

all: check build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/adag-lens

test:
	go test -count=1 ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || { echo "These files need gofmt:"; gofmt -l .; exit 1; }

check: fmt-check vet test

clean:
	rm -f $(BINARY) $(BINARY).exe
