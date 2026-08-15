# Security

Report vulnerabilities privately through GitHub Security Advisories for
`openmirai/clover-sdks`. Do not open a public issue with credentials, tokens,
customer payloads, or exploit details.

SDKs must reject invalid base URLs and idempotency keys before transport, avoid
cross-origin authorization redirects, cap response bodies, preserve structured
error metadata, and never log bearer tokens or request bodies by default.
