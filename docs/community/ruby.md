# Ruby integration guide (documentation-only)

This is a maintained integration guide, not a supported Clover Ruby SDK. The
former `clover-sdk-ruby` implementation was intentionally removed. Ruby teams
can call the REST API directly or contribute a separately owned community gem.

## Contract-first workflow

Generate a client from [`backend/api/openapi.json`](../../backend/api/openapi.json)
with a pinned OpenAPI Generator target, or wrap a reviewed HTTP library while
deriving request/response models from the schema. Keep generated output and
contract review in the consuming repository.

## Minimal REST shape

```ruby
uri = URI.join(base_url, "/v1/emails")
request = Net::HTTP::Post.new(uri)
request["Authorization"] = "Bearer #{api_key}"
request["Idempotency-Key"] = stable_key # 8-128 ASCII chars
request["Content-Type"] = "application/json"
request.body = JSON.generate(from: from, to: to, subject: subject, html: html)
response = Net::HTTP.start(uri.host, uri.port, use_ssl: uri.scheme == "https",
                           open_timeout: 10, read_timeout: 30) { |http| http.request(request) }
```

Use HTTPS in deployments, cap response reads, disable unsafe cross-host
redirects, and derive a stable idempotency key before sending. Reuse that key
for retries; batch items must not contain `scheduled_at`.

Successful bodies are JSON. Non-2xx bodies use RFC 9457-style
`application/problem+json`; preserve unknown members and `X-Request-ID`. Retry
only safe/idempotent operations and honor valid non-negative `Retry-After`,
including an explicit zero. Treat malformed or over-limit bodies as failures.

## Community target

RubyGems packaging, CI, and support are not provided by Clover for this guide.
A community gem must establish ownership, tests, bounded transport,
security/release policy, and a separate repository before official support is
considered.
