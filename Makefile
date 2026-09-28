.PHONY: build test lint clean install

BINARY=liegenpulse
VERSION?=0.1.0

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/$(BINARY) ./cmd/liegenpulse

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/

install:
	go install ./cmd/liegenpulse
