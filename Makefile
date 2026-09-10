.PHONY: build run test vet fmt schemas

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

schemas:
	./scripts/download-schemas.sh schemas
