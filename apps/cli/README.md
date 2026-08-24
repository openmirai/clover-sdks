# clover-cli

Dependency-free Clover Phase 1 command-line client. Set `CLOVER_BASE_URL` and
`CLOVER_API_KEY`, then select the clean account/environment scope with
`CLOVER_ACCOUNT_ID` and `CLOVER_ENVIRONMENT_ID` (or pass `-account-id` and
`-environment-id` on each command).

The platform commands use only the finalized routes under
`/api/v1/platform/accounts/{account_id}/environments/{environment_id}`. They
do not send the retired organization/project/tenant scope query or request
body fields.

## Transactional messages

```text
clover-cli send -account-id <account-id> -environment-id <environment-id> -json '{"from":{"address":"sender@example.com"},"to":[{"address":"user@example.com"}],"subject":"Hello","text":"Body"}' -idempotency-key send-1234
clover-cli messages send-batch -account-id <account-id> -environment-id <environment-id> -items '[{"subject":"Hello"}]' -idempotency-key batch-1234
clover-cli schedule <message-id> -account-id <account-id> -environment-id <environment-id> -scheduled-at 2030-01-01T00:00:00Z -idempotency-key schedule-1234
clover-cli get <message-id> -account-id <account-id> -environment-id <environment-id>
clover-cli list -account-id <account-id> -environment-id <environment-id> -status queued -limit 50
clover-cli cancel <message-id> -account-id <account-id> -environment-id <environment-id> -idempotency-key cancel-1234
```

Every JSON request accepts either `-json` or a bounded `-file`; `-file -`
reads stdin. The client rejects oversized/invalid JSON and validates required
idempotency keys before making a request. IDs are escaped as URL path
segments.

## Templates, webhooks, and timeline logs

```text
clover-cli templates list -account-id <account-id> -environment-id <environment-id>
clover-cli templates create -account-id <account-id> -environment-id <environment-id> -file template.json -idempotency-key template-1234
clover-cli templates versions create <template-id> -account-id <account-id> -environment-id <environment-id> -file version.json -idempotency-key version-1234
clover-cli templates versions publish <template-id> <version-ref> -account-id <account-id> -environment-id <environment-id> -idempotency-key publish-1234
clover-cli templates versions render <template-id> <version-ref> -account-id <account-id> -environment-id <environment-id> -json '{"name":"Ada"}'
clover-cli webhooks list -account-id <account-id> -environment-id <environment-id>
clover-cli webhooks create -account-id <account-id> -environment-id <environment-id> -file webhook.json -idempotency-key webhook-1234
clover-cli webhooks deliveries replay <delivery-id> -account-id <account-id> -environment-id <environment-id>
clover-cli logs -account-id <account-id> -environment-id <environment-id> -message-kind transactional
```

`logs` is the durable message-timeline query, not an assumed SSE endpoint.
Webhook delivery replay is a server-side operation and is scoped to the same
account/environment as the delivery.

## Inbound, preferences, suppressions, and SMTP

Provider callback ingress remains public and uses the verification headers:

```text
clover-cli inbound ses -file provider-event.json -provider-signature <signature> -provider-timestamp <unix-seconds> -provider-event-id <event-id>
clover-cli preferences topics list -account-id <account-id> -environment-id <environment-id>
clover-cli preferences topics create -account-id <account-id> -environment-id <environment-id> -file topic.json -idempotency-key topic-1234
clover-cli preferences token -account-id <account-id> -environment-id <environment-id> -file token.json
clover-cli suppressions list -account-id <account-id> -environment-id <environment-id> -active true
clover-cli suppressions reactivate <suppression-id> -account-id <account-id> -environment-id <environment-id>
clover-cli smtp credentials create -account-id <account-id> -environment-id <environment-id> -file credential.json -idempotency-key credential-1234
clover-cli smtp credentials rotate <credential-id> -account-id <account-id> -environment-id <environment-id> -idempotency-key rotate-1234
clover-cli smtp submissions list -account-id <account-id> -environment-id <environment-id> -status accepted
```

The callback command keeps provider signatures separate from bearer API-key
authentication and never retries a non-idempotent callback.

## Contacts, segments, automations, routing, and domain health

```text
clover-cli contacts list -account-id <account-id> -environment-id <environment-id>
clover-cli contacts create -account-id <account-id> -environment-id <environment-id> -file contact.json -idempotency-key contact-1234
clover-cli segments preview <segment-id> -account-id <account-id> -environment-id <environment-id>
clover-cli automations run <automation-id> -account-id <account-id> -environment-id <environment-id> -file run.json -idempotency-key run-1234
clover-cli routing policy get -account-id <account-id> -environment-id <environment-id>
clover-cli routing policy put -account-id <account-id> -environment-id <environment-id> -file policy.json -idempotency-key policy-1234
clover-cli routing pools list -account-id <account-id> -environment-id <environment-id>
clover-cli routing providers routes <provider-id> -account-id <account-id> -environment-id <environment-id> -file routes.json -idempotency-key routes-1234
clover-cli domain-health list -account-id <account-id> -environment-id <environment-id>
clover-cli domain-health check <domain-id> -account-id <account-id> -environment-id <environment-id>
clover-cli domain-health reports <domain-id> -account-id <account-id> -environment-id <environment-id> -limit 50
```

## Usage facts and reconciliation

Pricing is intentionally outside this CLI phase. The usage commands expose
the versioned vocabulary, facts, corrections, and reconciliation workflow so
the product is ready for later billing integration:

```text
clover-cli usage vocabulary -account-id <account-id> -environment-id <environment-id>
clover-cli usage facts -account-id <account-id> -environment-id <environment-id> -from 2030-01-01T00:00:00Z -to 2030-02-01T00:00:00Z
clover-cli usage export -account-id <account-id> -environment-id <environment-id> -fact-type message.accepted
clover-cli usage reconciliations create -account-id <account-id> -environment-id <environment-id> -file reconciliation.json
clover-cli usage reconciliations items <reconciliation-id> -account-id <account-id> -environment-id <environment-id>
clover-cli usage reconciliations items <reconciliation-id> add -account-id <account-id> -environment-id <environment-id> -file evidence.json
clover-cli usage reconciliations finish <reconciliation-id> -account-id <account-id> -environment-id <environment-id> -file result.json
```

## Accounts and environments

```text
clover-cli accounts list
clover-cli accounts get <account-id>
clover-cli environments list -account-id <account-id>
clover-cli environments create -account-id <account-id> -file environment.json -idempotency-key environment-1234
```

The client sends bearer authentication, a stable user agent, bounded request
and response bodies, and idempotency headers. It retries only GETs and
mutations that carry a required idempotency key, with at most three attempts.

## Verification

```text
go test ./...
go vet ./...
go test -race ./...
go build ./...
```

Unit tests use an injected round-tripper and never need a live service. To run
the opt-in deployed-backend check, set
`CLOVER_PHASE1_CONFORMANCE_URL`, `CLOVER_API_KEY`, `CLOVER_ACCOUNT_ID`, and
`CLOVER_ENVIRONMENT_ID`, then run:

```text
go test -run TestPlatformBackendConformance ./...
```

`dev` remains available as a local verified webhook receiver for integration
workflows.
