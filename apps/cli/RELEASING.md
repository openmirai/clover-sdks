# Releasing

Run `gofmt`, `go vet ./...`, `golangci-lint run ./...`, `go test -race ./...`,
and `goreleaser release --snapshot --clean`. Update `CHANGELOG.md`, create a
signed `apps/cli/vX.Y.Z` tag, and push it. GoReleaser publishes reproducible binaries
and checksums to the GitHub release.
