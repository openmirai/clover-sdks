import 'dart:async';
import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:http/http.dart' as http;

typedef JsonObject = Map<String, dynamic>;
typedef Sleep = Future<void> Function(Duration delay);

class EmailAddress {
  const EmailAddress({required this.address, this.name});

  final String address;
  final String? name;

  factory EmailAddress.fromJson(JsonObject json) => EmailAddress(
        address: json['address'] as String,
        name: json['name'] as String?,
      );

  JsonObject toJson() => <String, dynamic>{
        'address': address,
        if (name != null) 'name': name,
      };
}

class SendEmailRequest {
  const SendEmailRequest({
    required this.from,
    required this.to,
    required this.subject,
    this.cc,
    this.bcc,
    this.replyTo,
    this.html,
    this.text,
    this.attachments,
    this.headers,
    this.tags,
    this.scheduledAt,
  });

  final EmailAddress from;
  final List<EmailAddress> to;
  final List<EmailAddress>? cc;
  final List<EmailAddress>? bcc;
  final List<EmailAddress>? replyTo;
  final String subject;
  final String? html;
  final String? text;
  final List<JsonObject>? attachments;
  final Map<String, String>? headers;
  final Map<String, String>? tags;
  final String? scheduledAt;

  JsonObject toJson() => <String, dynamic>{
        'from': from.toJson(),
        'to': to.map((address) => address.toJson()).toList(growable: false),
        if (cc != null) 'cc': cc!.map((address) => address.toJson()).toList(growable: false),
        if (bcc != null) 'bcc': bcc!.map((address) => address.toJson()).toList(growable: false),
        if (replyTo != null) 'reply_to': replyTo!.map((address) => address.toJson()).toList(growable: false),
        'subject': subject,
        if (html != null) 'html': html,
        if (text != null) 'text': text,
        if (attachments != null) 'attachments': attachments,
        if (headers != null) 'headers': headers,
        if (tags != null) 'tags': tags,
        if (scheduledAt != null) 'scheduled_at': scheduledAt,
      };

  JsonObject toBatchJson() {
    final json = toJson();
    json.remove('scheduled_at');
    return json;
  }
}

class EmailSummary {
  const EmailSummary({required this.id, required this.status, this.scheduledAt, this.requestId, this.extra = const {}});

  final String id;
  final String status;
  final String? scheduledAt;
  final String? requestId;
  final JsonObject extra;

  factory EmailSummary.fromJson(JsonObject json) {
    return EmailSummary(
      id: json['id'] as String,
      status: json['status'] as String,
      scheduledAt: json['scheduled_at'] as String?,
      requestId: json['request_id'] as String?,
      extra: Map<String, dynamic>.from(json)
        ..removeWhere((key, _) => {'id', 'status', 'scheduled_at', 'request_id'}.contains(key)),
    );
  }
}

typedef EmailAccepted = EmailSummary;

class EmailBatchAccepted {
  const EmailBatchAccepted({required this.data, this.requestId, this.extra = const {}});

  final List<EmailSummary> data;
  final String? requestId;
  final JsonObject extra;

  factory EmailBatchAccepted.fromJson(JsonObject json) => EmailBatchAccepted(
        data: (json['data'] as List<dynamic>? ?? const [])
            .whereType<JsonObject>()
            .map(EmailSummary.fromJson)
            .toList(growable: false),
        requestId: json['request_id'] as String?,
        extra: Map<String, dynamic>.from(json)
          ..removeWhere((key, _) => {'data', 'request_id'}.contains(key)),
      );
}

class EmailPage {
  const EmailPage({required this.data, this.nextCursor, this.extra = const {}});

  final List<JsonObject> data;
  final String? nextCursor;
  final JsonObject extra;

  factory EmailPage.fromJson(JsonObject json) => EmailPage(
        data: (json['data'] as List<dynamic>? ?? const [])
            .whereType<JsonObject>()
            .map((item) => Map<String, dynamic>.from(item))
            .toList(growable: false),
        nextCursor: json['next_cursor'] as String?,
        extra: Map<String, dynamic>.from(json)
          ..removeWhere((key, _) => {'data', 'next_cursor'}.contains(key)),
      );
}

class EmailDetail {
  const EmailDetail(this.value);

  final JsonObject value;

  factory EmailDetail.fromJson(JsonObject json) => EmailDetail(Map<String, dynamic>.from(json));
}

class ListEmailsOptions {
  const ListEmailsOptions({
    this.cursor,
    this.limit,
    this.status,
    this.domainId,
    this.apiKeyId,
    this.requestId,
    this.createdAfter,
    this.createdBefore,
    this.additional = const {},
  });

  final String? cursor;
  final int? limit;
  final String? status;
  final String? domainId;
  final String? apiKeyId;
  final String? requestId;
  final String? createdAfter;
  final String? createdBefore;
  final Map<String, String> additional;

  Map<String, String> toQuery() => <String, String>{
        ...additional,
        if (cursor != null) 'cursor': cursor!,
        if (limit != null) 'limit': '$limit',
        if (status != null) 'status': status!,
        if (domainId != null) 'domain_id': domainId!,
        if (apiKeyId != null) 'api_key_id': apiKeyId!,
        if (requestId != null) 'request_id': requestId!,
        if (createdAfter != null) 'created_after': createdAfter!,
        if (createdBefore != null) 'created_before': createdBefore!,
      };
}

class ResponseMetadata {
  const ResponseMetadata({
    required this.statusCode,
    this.requestId,
    this.retryAfter,
    this.rateLimitRemaining,
    this.replayed = false,
  });

  final int statusCode;
  final String? requestId;
  final Duration? retryAfter;
  final int? rateLimitRemaining;
  final bool replayed;
}

class CloverResponse<T> {
  const CloverResponse({required this.value, required this.metadata});

  final T value;
  final ResponseMetadata metadata;
}

class ProblemDocument {
  const ProblemDocument({
    this.type,
    this.title,
    this.status,
    this.code,
    this.detail,
    this.requestId,
    this.fieldErrors,
    this.extra = const {},
  });

  final String? type;
  final String? title;
  final int? status;
  final String? code;
  final String? detail;
  final String? requestId;
  final Map<String, dynamic>? fieldErrors;
  final JsonObject extra;

  factory ProblemDocument.fromJson(JsonObject json) {
    const known = {'type', 'title', 'status', 'code', 'detail', 'request_id', 'field_errors'};
    final extra = Map<String, dynamic>.from(json)..removeWhere((key, _) => known.contains(key));
    return ProblemDocument(
      type: json['type'] as String?,
      title: json['title'] as String?,
      status: json['status'] as int?,
      code: json['code'] as String?,
      detail: json['detail'] as String?,
      requestId: json['request_id'] as String?,
      fieldErrors: (json['field_errors'] as Map?)?.cast<String, dynamic>(),
      extra: extra,
    );
  }
}

class CloverException implements Exception {
  CloverException({required this.statusCode, required this.metadata, this.problem, String? message})
      : message = message ?? problem?.title ?? 'Clover request failed ($statusCode)';

  final int statusCode;
  final ProblemDocument? problem;
  final ResponseMetadata metadata;
  final String message;

  @override
  String toString() => 'CloverException($statusCode): $message';
}

class _ResponseBodyLimitExceeded implements Exception {}

class CloverClient {
  static const defaultMaxResponseBodyBytes = 4 * 1024 * 1024;

  CloverClient({
    required String baseUrl,
    required this.apiKey,
    this.userAgent = 'clover-sdk-dart/0.1.0',
    http.Client? httpClient,
    int maxRetries = 2,
    int maxResponseBodyBytes = defaultMaxResponseBodyBytes,
    this.retryBaseDelay = const Duration(milliseconds: 100),
    Sleep? sleep,
  })  : assert(apiKey != ''),
        _baseUrl = _normalizeBaseUrl(baseUrl),
        httpClient = httpClient ?? http.Client(),
        sleep = sleep ?? ((delay) => Future<void>.delayed(delay)),
        maxRetries = maxRetries.clamp(0, 3).toInt(),
        maxResponseBodyBytes = maxResponseBodyBytes < 1 ? 1 : maxResponseBodyBytes {
    if (apiKey.trim().isEmpty) {
      throw ArgumentError.value(apiKey, 'apiKey', 'is required');
    }
  }

  final String _baseUrl;
  final String apiKey;
  final String userAgent;
  final http.Client httpClient;
  final int maxRetries;
  final int maxResponseBodyBytes;
  final Duration retryBaseDelay;
  final Sleep sleep;

  void close() => httpClient.close();

  static const _retryable = {408, 425, 429, 500, 502, 503, 504};

  Future<CloverResponse<EmailAccepted>> send(SendEmailRequest request, {required String idempotencyKey}) {
    return _request('/api/v1/emails', method: 'POST', body: request.toJson(), idempotencyKey: idempotencyKey, decode: EmailAccepted.fromJson);
  }

  Future<CloverResponse<EmailBatchAccepted>> sendBatch(List<SendEmailRequest> items, {required String idempotencyKey}) {
    return _request('/api/v1/emails/batch', method: 'POST', body: <String, dynamic>{'items': items.map((item) => item.toBatchJson()).toList(growable: false)}, idempotencyKey: idempotencyKey, decode: EmailBatchAccepted.fromJson);
  }

  Future<CloverResponse<EmailAccepted>> schedule(String emailId, String scheduledAt, {required String idempotencyKey}) {
    return _request('/api/v1/emails/${Uri.encodeComponent(emailId)}/schedule', method: 'POST', body: <String, dynamic>{'scheduled_at': scheduledAt}, idempotencyKey: idempotencyKey, decode: EmailAccepted.fromJson);
  }

  Future<CloverResponse<EmailSummary>> cancel(String emailId, {required String idempotencyKey}) {
    return _request('/api/v1/emails/${Uri.encodeComponent(emailId)}/cancel', method: 'POST', idempotencyKey: idempotencyKey, decode: EmailSummary.fromJson);
  }

  Future<CloverResponse<EmailDetail>> get(String emailId) {
    return _request('/api/v1/emails/${Uri.encodeComponent(emailId)}', decode: EmailDetail.fromJson);
  }

  Future<CloverResponse<EmailPage>> list({ListEmailsOptions options = const ListEmailsOptions()}) {
    return _request('/api/v1/emails', query: options.toQuery(), decode: EmailPage.fromJson);
  }

  Future<CloverResponse<T>> _request<T>(
    String path, {
    String method = 'GET',
    JsonObject? body,
    String? idempotencyKey,
    Map<String, String> query = const {},
    required T Function(JsonObject) decode,
  }) async {
    if (method != 'GET' && !_isValidIdempotencyKey(idempotencyKey)) {
      throw ArgumentError.value(idempotencyKey, 'idempotencyKey', 'must be 8-128 ASCII characters matching [A-Za-z0-9][A-Za-z0-9._:-]{7,127}');
    }
    final uri = _uri(path, query);
    final headers = <String, String>{
      'accept': 'application/json, application/problem+json',
      'authorization': 'Bearer $apiKey',
      'user-agent': userAgent,
      'x-request-id': _createRequestId(),
      if (body != null) 'content-type': 'application/json',
      if (idempotencyKey != null) 'idempotency-key': idempotencyKey,
    };
    for (var attempt = 0;; attempt++) {
      final request = http.Request(method, uri)..headers.addAll(headers);
      if (body != null) request.body = jsonEncode(body);
      final streamed = await httpClient.send(request);
      final metadata = _metadata(streamed.statusCode, streamed.headers);
      final responseBody = await _readBoundedBody(streamed, metadata);
      final decodedBody = _decodeBody(responseBody);
      if (streamed.statusCode >= 200 && streamed.statusCode < 300) {
        return CloverResponse(value: decode(_unwrapEnvelope(decodedBody)), metadata: metadata);
      }
      if ((method == 'GET' || idempotencyKey != null) && _retryable.contains(streamed.statusCode) && attempt < maxRetries) {
        await sleep(metadata.retryAfter ?? retryBaseDelay * (1 << attempt));
        continue;
      }
      final problem = _problemFromResponse(decodedBody, streamed.statusCode);
      throw CloverException(statusCode: streamed.statusCode, problem: problem, metadata: metadata);
    }
  }

  static Map<String, dynamic> _unwrapEnvelope(Map<String, dynamic> decoded) {
    if (decoded['success'] is! bool) return decoded;
    if (decoded['success'] != true) return decoded;
    final data = decoded['data'];
    if (data is Map<String, dynamic>) return data;
    return <String, dynamic>{};
  }

  Future<String> _readBoundedBody(http.StreamedResponse response, ResponseMetadata metadata) async {
    final bytes = BytesBuilder(copy: false);
    var total = 0;
    try {
      await for (final chunk in response.stream) {
        total += chunk.length;
        if (total > maxResponseBodyBytes) {
          throw _ResponseBodyLimitExceeded();
        }
        bytes.add(chunk);
      }
    } on _ResponseBodyLimitExceeded {
      throw CloverException(
        statusCode: response.statusCode,
        metadata: metadata,
        message: 'Clover response body exceeds the configured limit',
      );
    }
    return utf8.decode(bytes.takeBytes());
  }

  Uri _uri(String path, Map<String, String> query) {
    final encodedQuery = query.isEmpty ? '' : '?${Uri(queryParameters: query).query}';
    return Uri.parse('$_baseUrl$path$encodedQuery');
  }

  static JsonObject _decodeBody(String body) {
    if (body.isEmpty) return <String, dynamic>{};
    try {
      final value = jsonDecode(body);
      return value is JsonObject ? value : <String, dynamic>{'data': value};
    } on FormatException {
      return <String, dynamic>{'raw': body};
    }
  }

  static bool _isProblem(JsonObject value) =>
      value['type'] is String && value['title'] is String && value['status'] is int && value['code'] is String;

  static ProblemDocument? _problemFromResponse(JsonObject value, int statusCode) {
    if (_isProblem(value)) return ProblemDocument.fromJson(value);
    final error = value['error'];
    if (value['success'] == false && error is Map<String, dynamic>) {
      final message = error['message'];
      final type = error['type'];
      final code = error['code'];
      if (message is String && type is String && code != null) {
        return ProblemDocument(
          type: type,
          title: message,
          status: statusCode,
          code: code.toString(),
          detail: message,
          requestId: value['requestId'] as String?,
          extra: <String, dynamic>{'error': error},
        );
      }
    }
    return null;
  }

  static final Random _requestRandom = Random.secure();
  static String _createRequestId() {
    final bytes = List<int>.generate(16, (_) => _requestRandom.nextInt(256));
    final value = bytes.map((byte) => byte.toRadixString(16).padLeft(2, '0')).join();
    return 'req_$value';
  }

  static ResponseMetadata _metadata(int statusCode, Map<String, String> headers) {
    final normalized = <String, String>{for (final entry in headers.entries) entry.key.toLowerCase(): entry.value};
    final retry = int.tryParse(normalized['retry-after'] ?? '');
    return ResponseMetadata(
      statusCode: statusCode,
      requestId: normalized['x-request-id'],
      retryAfter: retry != null && retry >= 0 ? Duration(seconds: retry) : null,
      rateLimitRemaining: int.tryParse(normalized['x-ratelimit-remaining'] ?? ''),
      replayed: normalized['idempotency-replayed']?.toLowerCase() == 'true',
    );
  }

  static String _normalizeBaseUrl(String value) {
    final candidate = value.trim();
    final parsed = Uri.tryParse(candidate);
    if (parsed == null || !{'http', 'https'}.contains(parsed.scheme.toLowerCase()) || parsed.host.isEmpty || parsed.userInfo.isNotEmpty || parsed.hasQuery || parsed.hasFragment) {
      throw ArgumentError.value(value, 'baseUrl', 'must be an absolute http(s) URL without credentials, query, or fragment');
    }
    return candidate.replaceFirst(RegExp(r'/+$'), '');
  }

  static bool _isValidIdempotencyKey(String? value) {
    if (value == null || value.length < 8 || value.length > 128) return false;
    bool asciiAlphaNumeric(int codeUnit) =>
        codeUnit >= 0x30 && codeUnit <= 0x39 ||
        codeUnit >= 0x41 && codeUnit <= 0x5a ||
        codeUnit >= 0x61 && codeUnit <= 0x7a;
    if (!asciiAlphaNumeric(value.codeUnitAt(0))) return false;
    for (var index = 1; index < value.length; index++) {
      final codeUnit = value.codeUnitAt(index);
      if (!asciiAlphaNumeric(codeUnit) && !<int>[0x2e, 0x5f, 0x3a, 0x2d].contains(codeUnit)) return false;
    }
    return true;
  }
}
