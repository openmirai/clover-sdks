.PHONY: fmt vet lint test check

fmt:
	gofmt -w $$(find . -name '*.go' -print)

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test ./...

check: fmt vet lint test
