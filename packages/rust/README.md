# Clover Rust SDK

The crate exposes a dependency-light client for Rust 1.82 and newer. Verify
formatting, Clippy, tests, and package metadata with:

```bash
cargo fmt --check
cargo clippy --all-targets --all-features -- -D warnings
cargo test
cargo package --allow-dirty
```

See [`RELEASING.md`](RELEASING.md) and [`SECURITY.md`](SECURITY.md) for the
release and vulnerability policies.

Rust 2021 client for Clover with the small, well-tested `serde_json` codec.
`CloverClient` sends bearer authentication, a `User-Agent`, and requires an
idempotency key for mutations.
The default ureq/rustls transport supports production `http://` and `https://`
endpoints. Inject a `Transport` with `with_transport` for deterministic tests
or an application-owned HTTP stack.

```rust
use clover_sdk::{CloverClient, JsonValue};
let client = CloverClient::new("https://api.example.com", api_key);
let accepted = client.send(JsonValue::object([
    ("subject", "Hello".into()), ("text", "Accepted asynchronously".into()),
]), "order-1234")?;
```

`send`, `send_batch`, `schedule`, `cancel`, `get`, and `list` return a parsed
`JsonValue` that retains unknown fields. `CloverError` decodes problem JSON and
exposes request/replay/rate-limit/retry-after metadata. GETs and idempotent
mutations retry transient statuses at most three times. Tests inject a
transport and no network is used.

Run `cargo test`.
