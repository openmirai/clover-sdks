# Releasing

Run `gofmt -l .`, `go vet ./...`, `golangci-lint run ./...`, and
`go test -race ./...` locally. Update `CHANGELOG.md`, create a signed
`packages/go/vX.Y.Z` tag, and push it. The tag workflow creates the GitHub release; this
library is consumed as a Go module and does not require binary packaging.
