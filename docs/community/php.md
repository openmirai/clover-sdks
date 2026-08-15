# PHP / Laravel integration guide (documentation-only)

This is a maintained integration guide, not a supported Clover PHP SDK. The
former `clover-sdk-php` implementation (including its Laravel provider/facade)
was intentionally removed. Applications may use a PSR-compatible HTTP client;
Laravel wiring remains application-owned.

## Contract-first workflow

Use [`backend/api/openapi.json`](../../backend/api/openapi.json) with a pinned
OpenAPI generator (for example, JanePHP or OpenAPI Generator) to produce DTOs
and an API client. Review generated code and keep the generated schema aligned
with the canonical contract rather than hand-copying endpoint models.

## Minimal REST shape

```php
$response = $http->request('POST', $baseUrl.'/v1/emails', [
    'headers' => [
        'Authorization' => 'Bearer '.$apiKey,
        'Idempotency-Key' => $stableKey, // 8-128 ASCII chars
        'Accept' => 'application/json, application/problem+json',
    ],
    'json' => compact('from', 'to', 'subject', 'html'),
    'timeout' => 30,
    'allow_redirects' => false,
]);
```

Use HTTPS, bounded response streaming, and redirect handling that cannot leak a
bearer token to another host. Generate a stable idempotency key before a
mutation and reuse it on retry; batch items must omit `scheduled_at`.

Non-2xx responses are RFC 9457-style `application/problem+json`; preserve
unknown members and correlate with `X-Request-ID`. Retry only safe/idempotent
requests and honor valid non-negative `Retry-After`, including explicit zero.

## Community target

Laravel service providers, facades, config publishing, Composer packaging, and
framework support are community contribution targets, not Clover-supported
artifacts. A proposed package must have independent ownership, tests, bounded
transport, security/release policy, and a separate repository.
