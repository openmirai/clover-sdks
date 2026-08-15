# .NET integration guide (documentation-only)

This is a maintained integration guide, not a supported Clover .NET SDK. The
former `clover-sdk-dotnet` implementation was intentionally removed. .NET
applications and community contributors should use the canonical REST contract
until a separately owned SDK is proposed.

## Contract-first workflow

Treat [`backend/api/openapi.json`](../../backend/api/openapi.json) as the source
of truth. Generate a client with Kiota, NSwag, or another reviewed OpenAPI
generator, pin the generator version, and review the generated diff whenever
the contract changes. Do not hand-maintain endpoint DTOs that diverge from the
OpenAPI document.

## Minimal REST shape

```csharp
using var request = new HttpRequestMessage(HttpMethod.Post, "/v1/emails");
request.Headers.Authorization = new("Bearer", apiKey);
request.Headers.Add("Idempotency-Key", idempotencyKey); // 8-128 ASCII chars
request.Content = JsonContent.Create(new { from, to, subject, html });
using var response = await httpClient.SendAsync(request,
    HttpCompletionOption.ResponseHeadersRead, cancellationToken);
```

Use HTTPS in deployed environments, bounded response reads, explicit timeouts,
and no automatic cross-host redirects carrying bearer credentials. For a
mutation, generate a stable key before sending and reuse it for retries; do not
use a random value per attempt. Batch items must not include `scheduled_at`.

Successful responses are JSON. Error responses use RFC 9457-style
`application/problem+json`; preserve unknown members and correlate failures by
`X-Request-ID`. Retry only safe/idempotent operations and honor a valid
non-negative `Retry-After` value, including an explicit zero.

## Community target

This guide is the supported documentation surface for .NET contributors. A
future community package must add its own repository, tests, bounded transport,
security policy, release provenance, and ownership before being listed as an
official Clover SDK.
