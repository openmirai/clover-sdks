.PHONY: fmt vet lint test build release-check check

fmt:
	gofmt -w $$(find . -name '*.go' -print)

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test ./...

build:
	go build -trimpath -ldflags='-s -w' -o bin/clover ./

release-check:
	goreleaser release --snapshot --clean

check: fmt vet lint test build
