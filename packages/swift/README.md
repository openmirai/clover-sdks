# Clover Swift SDK

`CloverSDK` is a Swift Package Manager client for the Clover email API. It
supports Apple platforms and Linux, uses `async`/`await`, and keeps the
transport injectable for deterministic tests and application tracing.

Add `https://github.com/openmirai/clover-sdks` as a Swift package dependency
and select the `CloverSDK` product. Swift releases use root `vX.Y.Z` tags.

```swift
import CloverSDK

let client = CloverClient(configuration: .init(
    baseURL: URL(string: "https://api.example.com")!,
    apiKey: ProcessInfo.processInfo.environment["CLOVER_API_KEY"]!
))
let accepted = try await client.send(
    .init(from: .init(address: "sender@example.com"),
          to: [.init(address: "user@example.com")],
          subject: "Hello", text: "Queued by Clover"),
    idempotencyKey: "order-1234"
)
print(accepted.value.id, accepted.metadata.requestID ?? "")
```

Mutations require a non-empty idempotency key before a request reaches the
transport. GETs and idempotent mutations retry transient responses at most
three times, honoring `Retry-After`. `ProblemDocument.extra` preserves unknown
RFC 9457 extension fields, and every response exposes correlation metadata.
Responses are bounded by a 4 MiB default; configure `maxResponseBodyBytes` to
adjust the limit.

From the repository root, run `make check-swift`.
