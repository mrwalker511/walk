.PHONY: build install test test-int bench coverage lint clean release

build:
	go build -o bin/walk .

install:
	go install .

test:
	go test ./... -v -short

test-int:
	go test ./... -v -tags integration

bench:
	go test ./... -bench=. -benchmem

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

lint:
	golangci-lint run

clean:
	rm -rf bin/ coverage.out

release:
	goreleaser release --clean
