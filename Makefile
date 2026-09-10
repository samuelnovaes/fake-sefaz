.PHONY: build run test vet fmt

build:
	go build -o bin/fake-sefaz ./cmd/fakesefaz

run:
	go run ./cmd/fakesefaz

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .
