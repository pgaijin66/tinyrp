## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## run: start the proxy server
.PHONY: run
run:
	go run cmd/main.go

## build: compile binary to ./bin/tinyrp
.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/tinyrp cmd/main.go

## test: run all tests with race detector
.PHONY: test
test:
	go test -race ./...

## bench: run benchmarks
.PHONY: bench
bench:
	go test -bench=. -benchmem -count=3 -cpu=1,4,8 ./bench/

## bench-quick: run benchmarks (single pass)
.PHONY: bench-quick
bench-quick:
	go test -bench=. -benchmem -count=1 -cpu=4 ./bench/

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## vet: run static analysis
.PHONY: vet
vet:
	go vet ./...
