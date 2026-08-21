import 'dart:async';
import 'dart:convert';

import 'package:clover_sdk/clover_sdk.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

void main() {
  test('sends auth headers, encodes path segments, and retries', () async {
    var attempts = 0;
    final client = CloverClient(
      baseUrl: 'https://api.example.test',
      apiKey: 're_test',
      maxRetries: 1,
      sleep: (_) async {},
      httpClient: MockClient((request) async {
        attempts += 1;
        expect(request.headers['authorization'], 'Bearer re_test');
        expect(request.headers['user-agent'], 'clover-sdk-dart/0.1.0');
        expect(request.headers['idempotency-key'], 'idem-1234');
        expect(request.url.toString(), contains('/api/v1/emails/a%2Fb/schedule'));
        if (attempts == 1) return http.Response('', 503, headers: {'retry-after': '0'});
        return http.Response(jsonEncode({'id': 'email-1', 'status': 'queued', 'request_id': 'req-1'}), 202, headers: {'x-request-id': 'req-1'});
      }),
    );

    final response = await client.schedule('a/b', '2026-08-15T00:00:00Z', idempotencyKey: 'idem-1234');
    expect(response.value.id, 'email-1');
    expect(response.metadata.requestId, 'req-1');
    expect(attempts, 2);
  });

  test('validates mutation idempotency before invoking client', () async {
    var called = false;
    final client = CloverClient(
      baseUrl: 'https://api.example.test',
      apiKey: 'key',
      httpClient: MockClient((_) async {
        called = true;
        return http.Response('', 500);
      }),
    );
    await expectLater(client.cancel('email-1', idempotencyKey: ' '), throwsArgumentError);
    expect(called, isFalse);
  });

  test('rejects idempotency keys outside the API contract', () async {
    final client = CloverClient(
      baseUrl: 'https://api.example.test',
      apiKey: 'key',
      httpClient: MockClient((_) async => http.Response('', 500)),
    );

    for (final key in <String>['short-1', 'starts with space', List<String>.filled(129, 'a').join(), 'starts#bad', '${List<String>.filled(8, 'a').join()}\n', '${List<String>.filled(8, 'a').join()}\r', '${List<String>.filled(8, 'a').join()}\u0000']) {
      await expectLater(client.cancel('email-1', idempotencyKey: key), throwsArgumentError);
    }
  });

  test('requires an absolute http(s) base URL', () {
    expect(() => CloverClient(baseUrl: 'ftp://api.example.test', apiKey: 'key'), throwsArgumentError);
    expect(() => CloverClient(baseUrl: '/relative', apiKey: 'key'), throwsArgumentError);
    expect(() => CloverClient(baseUrl: 'https://user:pass@api.example.test', apiKey: 'key'), throwsArgumentError);
  });

  test('preserves unknown problem extensions and metadata', () async {
    final client = CloverClient(
      baseUrl: 'https://api.example.test',
      apiKey: 'key',
      httpClient: MockClient((_) async => http.Response(
            jsonEncode({'type': 'https://example.test/problem', 'title': 'Nope', 'status': 422, 'code': 'invalid', 'new_extension': {'flag': true}}),
            422,
            headers: {'x-request-id': 'req-2', 'x-ratelimit-remaining': '7'},
          )),
    );

    await expectLater(client.get('email-1'), throwsA(isA<CloverException>().having((error) => error.problem!.extra['new_extension']['flag'], 'extension', true)));
  });

  test('omits schedule-only fields from batch items', () async {
    final client = CloverClient(
      baseUrl: 'https://api.example.test',
      apiKey: 'key',
      httpClient: MockClient((request) async {
        final body = jsonDecode(request.body) as Map<String, dynamic>;
        final item = (body['items'] as List<dynamic>).single as Map<String, dynamic>;
        expect(item.containsKey('scheduled_at'), isFalse);
        return http.Response(jsonEncode({'data': <dynamic>[], 'request_id': 'req-1'}), 202);
      }),
    );
    await client.sendBatch(
      [
        const SendEmailRequest(
          from: EmailAddress(address: 'sender@example.com'),
          to: [EmailAddress(address: 'user@example.com')],
          subject: 'Hello',
          text: 'Queued',
          scheduledAt: '2026-08-15T00:00:00Z',
        ),
      ],
      idempotencyKey: 'idem-1234',
    );
  });

  test('response body limits preserve status and metadata', () async {
    final client = CloverClient(
      baseUrl: 'https://api.example.test',
      apiKey: 'key',
      maxResponseBodyBytes: 8,
      httpClient: MockClient((_) async => http.Response(List<String>.filled(32, 'x').join(), 503, headers: {'x-request-id': 'req-large'})),
    );
    await expectLater(
      client.get('email-1'),
      throwsA(isA<CloverException>().having((error) => error.statusCode, 'status', 503).having((error) => error.metadata.requestId, 'request id', 'req-large')),
    );
  });

  test('streaming response body limits stop before decode', () async {
    final client = CloverClient(
      baseUrl: 'https://api.example.test',
      apiKey: 'key',
      maxResponseBodyBytes: 8,
      httpClient: _StreamingClient(),
    );
    await expectLater(
      client.get('email-1'),
      throwsA(isA<CloverException>()
          .having((error) => error.statusCode, 'status', 503)
          .having((error) => error.message, 'message', contains('body'))),
    );
  });

  test('successful response models preserve unknown fields', () async {
    final client = CloverClient(
      baseUrl: 'https://api.example.test',
      apiKey: 'key',
      httpClient: MockClient((_) async => http.Response(
            jsonEncode({'id': 'email-1', 'status': 'queued', 'request_id': 'req-1', 'vendor': {'flag': true}}),
            202,
          )),
    );
    final response = await client.send(
      const SendEmailRequest(
        from: EmailAddress(address: 'sender@example.com'),
        to: [EmailAddress(address: 'user@example.com')],
        subject: 'Hello',
        text: 'Queued',
      ),
      idempotencyKey: 'idem-1234',
    );
    expect(response.value.extra['vendor']['flag'], isTrue);
  });
}

class _StreamingClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    return http.StreamedResponse(
      Stream<List<int>>.fromIterable([List<int>.filled(32, 120)]),
      503,
      headers: const {'x-request-id': 'req-stream-large'},
    );
  }
}
