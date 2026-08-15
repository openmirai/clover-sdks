# Clover Java SDK

Dependency-free Java 17+ client for Clover. `CloverClient` uses
`java.net.http.HttpClient`, bearer authentication, and a `User-Agent`; every
mutation requires an idempotency key.

```java
var client = new CloverClient("https://api.example.com", System.getenv("CLOVER_API_KEY"));
var accepted = client.send(Map.of(
    "from", Map.of("address", "sender@example.com"),
    "to", List.of(Map.of("address", "user@example.com")),
    "subject", "Hello", "text", "Accepted asynchronously"), "order-1234");
```

`send`, `sendBatch`, `schedule`, `cancel`, `get`, and `list` return map-shaped
JSON while retaining unknown members. `CloverException` decodes problem JSON
and includes request/replay/rate-limit/retry-after metadata. Safe requests retry
transient statuses at most three times. Inject `Transport` and a sleeper for
deterministic tests.

Build and verify with `mvn verify`; Checkstyle, SpotBugs, and the JUnit 5
conformance test run as part of the build. Releases use the Maven Release
Plugin/tag workflow described in [`RELEASING.md`](RELEASING.md).

Kotlin/JVM applications use the same Maven artifact and Java API directly:

```kotlin
val client = CloverClient("https://api.example.com", System.getenv("CLOVER_API_KEY"))
val accepted = client.send(mapOf<String, Any>("subject" to "Hello"), "order-1234")
```
