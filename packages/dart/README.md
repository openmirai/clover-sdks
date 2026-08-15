# Clover Dart SDK

`clover_sdk` is a typed asynchronous Dart client for the Clover email API. It
works in Dart and Flutter, uses `package:http` by default, and accepts an
injected `http.Client` (or `MockClient`) for deterministic tests and tracing.

```dart
import 'package:clover_sdk/clover_sdk.dart';

final client = CloverClient(
  baseUrl: 'https://api.example.com',
  apiKey: const String.fromEnvironment('CLOVER_API_KEY'),
);
final accepted = await client.send(
  const SendEmailRequest(
    from: EmailAddress(address: 'sender@example.com'),
    to: [EmailAddress(address: 'user@example.com')],
    subject: 'Hello',
    text: 'Queued by Clover',
  ),
  idempotencyKey: 'order-1234',
);
print(accepted.value.id);
```

The client exposes `send`, `sendBatch`, `schedule`, `cancel`, `get`, and
`list`. Mutation idempotency keys are checked before transport invocation.
GETs and idempotent mutations retry transient statuses at most three times and
honor `Retry-After`. Problem responses preserve unknown extension fields in
`ProblemDocument.extra`, and successful/error responses expose correlation
metadata.
Response bodies are bounded by a 4 MiB default; configure
`maxResponseBodyBytes` to adjust the limit.
Call `client.close()` when the client is no longer needed.

Run `dart format --output=none --set-exit-if-changed .`, `dart analyze`,
`dart test`, and `dart pub publish --dry-run` locally.
