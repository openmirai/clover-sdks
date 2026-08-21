# clover-cli

Dependency-free Clover command-line client. Set `CLOVER_BASE_URL` and
`CLOVER_API_KEY`, then use:

```text
clover-cli send -from sender@example.com -to user@example.com -subject Hello -text Body -idempotency-key order-1234
clover-cli send-batch -items '[{"subject":"Hello"}]' -idempotency-key batch-1234
clover-cli schedule <email-id> -scheduled-at 2030-01-01T00:00:00Z -idempotency-key schedule-1234
clover-cli get <email-id>
clover-cli list -status queued -limit 50 -cursor next-page
clover-cli cancel <email-id> -idempotency-key cancel-1234
clover-cli domains list -status verified -limit 50
clover-cli domains get <domain-id>
clover-cli domains create -file domain.json -idempotency-key domain-1234
clover-cli domains configure <domain-id> -json '{"sendingEnabled":true}' -idempotency-key config-1234
clover-cli domains verify <domain-id>
clover-cli domains dns-records <domain-id>
clover-cli domains provision <domain-id> -idempotency-key provision-1234
clover-cli domain-health list -project-id <project-id> -environment staging
clover-cli templates list -project-id <project-id> -environment staging -tenant-id <tenant-id>
clover-cli templates create -file template.json -organization-id <organization-id> -project-id <project-id> -environment staging -tenant-id <tenant-id> -idempotency-key template-1234
clover-cli templates versions list <template-id> -project-id <project-id> -environment staging -tenant-id <tenant-id>
clover-cli templates versions get <template-id> latest -project-id <project-id> -environment staging -tenant-id <tenant-id>
clover-cli templates versions compare <template-id> -from 1 -to latest -project-id <project-id> -environment staging -tenant-id <tenant-id>
clover-cli templates versions create <template-id> -file version.json -organization-id <organization-id> -project-id <project-id> -environment staging -tenant-id <tenant-id> -idempotency-key version-1234
clover-cli templates publish <template-id> <version-id> -project-id <project-id> -environment staging -tenant-id <tenant-id> -idempotency-key publish-1234
clover-cli templates versions rollback <template-id> <version-id> -project-id <project-id> -environment staging -tenant-id <tenant-id> -idempotency-key rollback-1234
clover-cli webhooks list -limit 50
clover-cli webhooks create -file webhook.json -idempotency-key webhook-1234
clover-cli webhooks rotate <webhook-id> -idempotency-key rotate-1234
clover-cli webhooks deliveries list -limit 50
clover-cli webhooks deliveries replay <delivery-id>
clover-cli api-keys list
clover-cli api-keys create -file api-key.json
clover-cli received list -limit 50
clover-cli inbound ses -file inbound.json -provider-signature <signature> -provider-timestamp <unix-seconds> -provider-event-id <event-id>
clover-cli provider-events cloudflare -file event.json -provider-signature <signature> -provider-timestamp <unix-seconds> -provider-event-id <event-id>
clover-cli preferences get <preference-token>
clover-cli suppressions list -active true
clover-cli contacts list -project-id <project-id> -environment staging -tenant-id <tenant-id>
clover-cli segments create -file segment.json -organization-id <organization-id> -project-id <project-id> -environment staging -tenant-id <tenant-id> -idempotency-key segment-1234
clover-cli broadcasts schedule <broadcast-id> -file schedule.json -project-id <project-id> -environment staging -tenant-id <tenant-id> -idempotency-key broadcast-1234
clover-cli automations run <automation-id> -file run.json -project-id <project-id> -environment staging -tenant-id <tenant-id> -idempotency-key run-1234
clover-cli automations update <automation-id> -file automation.json -organization-id <organization-id> -project-id <project-id> -environment staging -tenant-id <tenant-id> -idempotency-key automation-update-1234
clover-cli audit list -project-id <project-id> -environment staging
clover-cli emails trace <email-id> -project-id <project-id> -environment staging
clover-cli delivery-policies list -project-id <project-id> -environment staging
clover-cli delivery-routes create -file route.json -organization-id <organization-id> -project-id <project-id> -environment staging -idempotency-key route-1234
clover-cli smtp-credentials list -project-id <project-id> -environment staging
clover-cli routing policy get
clover-cli routing policy put -file routing-policy.json -idempotency-key policy-1234
clover-cli routing capabilities
clover-cli routing pools list
clover-cli routing pools create -file pool.json -idempotency-key pool-1234
clover-cli routing pools warmup <pool-id> -idempotency-key warmup-1234
clover-cli routing pools command <pool-id> -file pool-command.json -idempotency-key command-1234
clover-cli routing pools ip-command <pool-id> <ip-id> -file ip-command.json -idempotency-key ip-command-1234
clover-cli routing audit -entity-type dedicated_pool
clover-cli logs -operation send -limit 50
clover-cli logs -follow -interval 2s
clover-cli dev --listen 127.0.0.1:8788 --path /webhooks/clover
```

`dev` starts a local webhook receiver and prints each verified JSON event as one
line. Set `CLOVER_WEBHOOK_SECRET` or pass `--secret`; the receiver verifies the
`X-Clover-Webhook-*` HMAC headers, rejects stale timestamps and duplicate
delivery IDs, and fails closed on malformed or oversized bodies. The Phase 1
resource commands match the backend Swagger paths and accept JSON either with
`-json` or a bounded `-file`/stdin input. Idempotency is required only where
the backend route/handler requires it (for example email sends, DNS
provisioning, templates, audience lifecycle transitions, automation, audit,
and email replay); domain CRUD, webhooks, preferences, suppressions, inbound,
provider-event, and API-key routes do not invent a requirement.

Every scoped lifecycle, template, contact, segment, broadcast, automation,
domain-health, audit, and email trace/replay command requires the scope flags
appropriate to the backend: `-project-id` and `-environment`, plus
`-tenant-id` where the route requires a tenant. The environment is also sent
as `X-Environment` because the backend accepts that header. Template,
contact, segment, broadcast, and automation create JSON bodies are populated
with the selected `scope` object. Set
`CLOVER_PROJECT_ID`, `CLOVER_ENVIRONMENT`, and `CLOVER_TENANT_ID` to avoid
repeating the flags. Body-backed scope mutations also require
`CLOVER_ORGANIZATION_ID` (or `-organization-id`) because the backend validates
the organization boundary inside the JSON scope object.

Template version `get`, `compare`, `publish`, and `rollback` use the exact
immutable-version paths. DNS `provision` uses the provider onboarding route;
the separate `domain-health` commands use the scoped health service. The
provider-neutral `routing` commands target the current routing policy,
capabilities, dedicated-pool, pool/IP lifecycle-command, and audit paths. Pool
creation carries the documented warm-up plan in its JSON body, while
`routing pools warmup` is a convenience for the `start_warmup` lifecycle
command; all routing mutations require the backend's `Idempotency-Key`.

The inbound and provider-event commands target public provider callback routes.
When callback verification is enabled, pass all three provider headers together
(or set `CLOVER_PROVIDER_SIGNATURE`, `CLOVER_PROVIDER_TIMESTAMP`, and
`CLOVER_PROVIDER_EVENT_ID`); the CLI keeps them separate from bearer API-key
authentication and never retries these non-idempotent callbacks.

`logs -follow` is bounded polling over the documented `GET /api/v1/logs`
endpoint. It sends the cursor returned in `X-Next-Cursor`, polls at the chosen
interval (minimum 100ms), suppresses duplicate log IDs/request IDs, writes one
compact JSON object per line, and exits on Ctrl-C. It does not assume an SSE
endpoint. `stream` is retained as an alias for `logs -follow`.

The shared API client sends bearer auth, user-agent, and idempotency headers,
decodes problem JSON while retaining unknown fields, bounds request/response
bodies, and retries only safe requests or idempotent mutations with a maximum
of three attempts. Tests inject an HTTP round-tripper; no real network is
needed.

Build or test with `go test ./...` and install with
`go install github.com/openmirai/clover-sdks/apps/cli@latest`.
