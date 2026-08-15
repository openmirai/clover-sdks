# Releasing

Run `cargo fmt --all --check`, `cargo clippy --all-targets --all-features --
-D warnings`, `cargo test --all-targets --all-features`, and `cargo package`
from a clean checkout. Update `CHANGELOG.md`, create and push a signed
`rust/vX.Y.Z` tag. The tag workflow publishes to crates.io with a scoped token and
creates the GitHub release.
