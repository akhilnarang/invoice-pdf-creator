BINARY := invoice-pdf-creator
VERSION ?= 0.1.0

.PHONY: build release test bench

build:
	go build -o $(BINARY) .

release:
	go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) .

test:
	go test ./...

bench:
	go test -run='^$$' -bench=. -benchmem ./...
