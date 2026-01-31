VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: test lint build all clean

test:
	go test -race ./...

lint:
	golangci-lint run

build:
	go build -ldflags "$(LDFLAGS)" -o ralph-loop ./cmd/ralph-loop/

all: test lint build

clean:
	rm -f ralph-loop coverage.out
