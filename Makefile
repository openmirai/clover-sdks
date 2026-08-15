.PHONY: check check-layout check-typescript check-python check-go check-java check-rust check-swift check-dart check-cli

check: check-layout check-typescript check-python check-go check-java check-rust check-swift check-dart check-cli

check-layout:
	python3 scripts/check_repository.py

check-typescript:
	npm --prefix packages/typescript ci --ignore-scripts
	npm --prefix packages/typescript run lint
	npm --prefix packages/typescript run format:check
	npm --prefix packages/typescript run typecheck
	npm --prefix packages/typescript run test:coverage

check-python:
	python3 -m pip install -e 'packages/python[dev]'
	cd packages/python && python3 -m ruff check . && python3 -m ruff format --check . && python3 -m mypy && python3 -m pytest --cov

check-go:
	cd packages/go && gofmt -w . && go vet ./... && go test -race ./...

check-java:
	cd packages/java && mvn -B -ntp verify

check-rust:
	cargo fmt --manifest-path packages/rust/Cargo.toml --all --check
	cargo clippy --manifest-path packages/rust/Cargo.toml --all-targets --all-features -- -D warnings
	cargo test --manifest-path packages/rust/Cargo.toml --all-targets --all-features

check-swift:
	swift test
	swift-format lint --strict --configuration .swift-format packages/swift/Sources packages/swift/Tests
	swiftlint lint --strict --no-cache --config .swiftlint.yml packages/swift/Sources packages/swift/Tests

check-dart:
	cd packages/dart && dart pub get && dart format --output=none --set-exit-if-changed . && dart analyze && dart test && dart pub publish --dry-run

check-cli:
	cd apps/cli && gofmt -w . && go vet ./... && go test -race ./... && go build -trimpath ./...
