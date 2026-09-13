BINARY  := ne-image-sorter
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: all build run test cover vet fmt lint clean install

all: vet test build

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/$(BINARY)

run: build
	./bin/$(BINARY)

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

vet:
	go vet ./...

fmt:
	gofmt -l -w .

lint: vet
	gofmt -l . | tee /dev/stderr | test ! -s /dev/stdin

install: build
	install -Dm755 bin/$(BINARY) $(HOME)/.local/bin/$(BINARY)

clean:
	rm -rf bin coverage.out
