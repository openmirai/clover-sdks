package dev.clover.sdk;

import java.io.IOException;
import java.io.InputStream;
import java.net.URI;
import java.net.http.*;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.*;

public final class CloverClient {
  public record ResponseMeta(String requestId, Long retryAfterSeconds, Long rateLimitRemaining, boolean replayed) {}
  public record RawResponse(int status, Map<String, String> headers, String body) {
    public RawResponse {
      headers = Map.copyOf(headers);
    }
  }
  @FunctionalInterface public interface Transport { RawResponse send(String method, String url, Map<String, String> headers, String body) throws IOException, InterruptedException; }

  public static final class CloverException extends RuntimeException {
    private static final long serialVersionUID = 1L;
    private final int status; private final Map<String, Object> problem; private final ResponseMeta metadata;
    CloverException(int status, String message, Map<String, Object> problem, ResponseMeta metadata) {
      super(message);
      this.status = status;
      this.problem = problem == null ? null : Collections.unmodifiableMap(new LinkedHashMap<>(problem));
      this.metadata = metadata;
    }
    public int status() { return status; } public Map<String, Object> problem() { return problem; } public ResponseMeta metadata() { return metadata; }
  }

  private static final Set<Integer> RETRYABLE = Set.of(408, 425, 429, 500, 502, 503, 504);
  public static final int DEFAULT_MAX_RESPONSE_BODY_BYTES = 4 * 1024 * 1024;
  private static final class ResponseBodyLimitException extends IOException {
    private static final long serialVersionUID = 1L;
    private final int status; private final Map<String, String> headers;
    ResponseBodyLimitException(int status, Map<String, String> headers) { super("Clover response body exceeds the configured limit"); this.status = status; this.headers = Map.copyOf(headers); }
  }
  private final String baseUrl; private final String apiKey; private final String userAgent; private final int maxRetries; private final long retryBaseDelayMillis; private final Transport transport; private final java.util.function.LongConsumer sleeper; private final int maxResponseBodyBytes; private final HttpClient httpClient;

  public CloverClient(String baseUrl, String apiKey) { this(baseUrl, apiKey, "clover-sdk-java/0.1.0", 2, 100, null, null, DEFAULT_MAX_RESPONSE_BODY_BYTES); }
  public CloverClient(String baseUrl, String apiKey, String userAgent, int maxRetries, long retryBaseDelayMillis, Transport transport, java.util.function.LongConsumer sleeper) {
    this(baseUrl, apiKey, userAgent, maxRetries, retryBaseDelayMillis, transport, sleeper, DEFAULT_MAX_RESPONSE_BODY_BYTES);
  }
  public CloverClient(String baseUrl, String apiKey, String userAgent, int maxRetries, long retryBaseDelayMillis, Transport transport, java.util.function.LongConsumer sleeper, int maxResponseBodyBytes) {
    URI parsed;
    String normalizedBaseUrl = baseUrl == null ? null : baseUrl.trim();
    try { parsed = normalizedBaseUrl == null ? null : URI.create(normalizedBaseUrl); } catch (IllegalArgumentException error) { parsed = null; }
    if (parsed == null || parsed.getHost() == null || parsed.getUserInfo() != null || parsed.getQuery() != null || parsed.getFragment() != null || !("http".equalsIgnoreCase(parsed.getScheme()) || "https".equalsIgnoreCase(parsed.getScheme()))) throw new IllegalArgumentException("baseUrl must be an absolute http(s) URL without userinfo/query/fragment");
    if (apiKey == null || apiKey.isBlank()) throw new IllegalArgumentException("apiKey is required");
    this.baseUrl = normalizedBaseUrl.replaceAll("/$", ""); this.apiKey = apiKey; this.userAgent = userAgent; this.maxRetries = Math.max(0, Math.min(3, maxRetries)); this.retryBaseDelayMillis = Math.max(0, retryBaseDelayMillis); this.maxResponseBodyBytes = Math.max(1, maxResponseBodyBytes); this.httpClient = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(30)).build(); this.transport = transport != null ? transport : this::httpTransport; this.sleeper = sleeper != null ? sleeper : milliseconds -> { try { Thread.sleep(milliseconds); } catch (InterruptedException error) { Thread.currentThread().interrupt(); } };
  }

  public Map<String, Object> send(Map<String, Object> request, String idempotencyKey) throws IOException, InterruptedException { return request("POST", "/api/v1/emails", request, idempotencyKey); }
  public Map<String, Object> sendBatch(List<Map<String, Object>> items, String idempotencyKey) throws IOException, InterruptedException { return request("POST", "/api/v1/emails/batch", Map.of("items", items.stream().map(item -> { Map<String, Object> copy = new LinkedHashMap<>(item); copy.remove("scheduled_at"); return copy; }).toList()), idempotencyKey); }
  public Map<String, Object> schedule(String id, String scheduledAt, String idempotencyKey) throws IOException, InterruptedException { return request("POST", "/api/v1/emails/" + pathSegment(id) + "/schedule", Map.of("scheduled_at", scheduledAt), idempotencyKey); }
  public Map<String, Object> cancel(String id, String idempotencyKey) throws IOException, InterruptedException { return request("POST", "/api/v1/emails/" + pathSegment(id) + "/cancel", null, idempotencyKey); }
  public Map<String, Object> get(String id) throws IOException, InterruptedException { return request("GET", "/api/v1/emails/" + pathSegment(id), null, null); }
  public Map<String, Object> list(Map<String, String> filters) throws IOException, InterruptedException { StringBuilder query = new StringBuilder(); if (filters != null) for (Map.Entry<String, String> entry : filters.entrySet()) query.append(query.length() == 0 ? '?' : '&').append(java.net.URLEncoder.encode(entry.getKey(), java.nio.charset.StandardCharsets.UTF_8)).append('=').append(java.net.URLEncoder.encode(entry.getValue(), java.nio.charset.StandardCharsets.UTF_8)); return request("GET", "/api/v1/emails" + query, null, null); }

  private Map<String, Object> request(String method, String path, Map<String, Object> payload, String idempotencyKey) throws IOException, InterruptedException {
    if (!method.equals("GET") && !validIdempotencyKey(idempotencyKey)) throw new IllegalArgumentException("idempotency key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$");
    Map<String, String> headers = new LinkedHashMap<>(); headers.put("Accept", "application/json, application/problem+json"); headers.put("Authorization", "Bearer " + apiKey); headers.put("User-Agent", userAgent); if (payload != null) headers.put("Content-Type", "application/json"); if (idempotencyKey != null) headers.put("Idempotency-Key", idempotencyKey);
    String body = payload == null ? null : JsonCodec.stringify(payload);
    for (int attempt = 0; ; attempt++) {
      RawResponse response;
      try {
        response = transport.send(method, baseUrl + path, headers, body);
      } catch (ResponseBodyLimitException error) {
        throw new CloverException(error.status, error.getMessage(), null, metadata(error.headers));
      }
      ResponseMeta meta = metadata(response.headers());
      if (response.body().getBytes(StandardCharsets.UTF_8).length > maxResponseBodyBytes) throw new CloverException(response.status(), "Clover response body exceeds the configured limit", null, meta);
      Map<String, Object> decoded;
      try {
        decoded = JsonCodec.object(response.body());
      } catch (RuntimeException error) {
        Map<String, Object> problem = Map.of("raw", response.body());
        throw new CloverException(response.status(), "Clover returned malformed JSON", problem, meta);
      }
      if (response.status() >= 200 && response.status() < 300) return unwrapEnvelope(decoded);
      if ((method.equals("GET") || idempotencyKey != null) && RETRYABLE.contains(response.status()) && attempt < maxRetries) { long delay = meta.retryAfterSeconds() == null ? retryBaseDelayMillis * (1L << attempt) : meta.retryAfterSeconds() * 1000; sleeper.accept(delay); continue; }
      if (Boolean.FALSE.equals(decoded.get("success")) && decoded.get("error") instanceof Map<?, ?> errorMap) {
        Object message = errorMap.get("message");
        throw new CloverException(response.status(), message == null ? "Clover request failed (" + response.status() + ")" : String.valueOf(message), decoded, meta);
      }
      Map<String, Object> problem = decoded.containsKey("title") && decoded.containsKey("type") ? decoded : null; throw new CloverException(response.status(), problem == null ? "Clover request failed (" + response.status() + ")" : String.valueOf(problem.get("title")), problem, meta);
    }
  }

  @SuppressWarnings("unchecked")
  private static Map<String, Object> unwrapEnvelope(Map<String, Object> decoded) {
    if (!(decoded.get("success") instanceof Boolean)) return decoded;
    if (!Boolean.TRUE.equals(decoded.get("success"))) return decoded;
    Object data = decoded.get("data");
    if (data instanceof Map<?, ?> map) return (Map<String, Object>) map;
    return Map.of();
  }

  private ResponseMeta metadata(Map<String, String> headers) { Map<String, String> lower = new HashMap<>(); headers.forEach((key, value) -> lower.put(key.toLowerCase(Locale.ROOT), value)); Long retry = number(lower.get("retry-after")); Long remaining = number(lower.get("x-ratelimit-remaining")); return new ResponseMeta(lower.get("x-request-id"), retry, remaining, Boolean.parseBoolean(lower.get("idempotency-replayed"))); }
  private static Long number(String value) { try { if (value == null) return null; long number = Long.parseLong(value); return number < 0 ? null : number; } catch (NumberFormatException error) { return null; } }
  private static boolean validIdempotencyKey(String value) {
    if (value == null || value.length() < 8 || value.length() > 128) return false;
    if (!asciiAlphaNumeric(value.charAt(0))) return false;
    for (int index = 1; index < value.length(); index++) {
      char character = value.charAt(index);
      if (!asciiAlphaNumeric(character) && character != '.' && character != '_' && character != ':' && character != '-') return false;
    }
    return true;
  }
  private static boolean asciiAlphaNumeric(char character) { return character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z' || character >= '0' && character <= '9'; }
  private static String pathSegment(String value) { if (value == null || value.isEmpty()) throw new IllegalArgumentException("email id is required"); return java.net.URLEncoder.encode(value, java.nio.charset.StandardCharsets.UTF_8).replace("+", "%20"); }
  private RawResponse httpTransport(String method, String url, Map<String, String> headers, String body) throws IOException, InterruptedException { HttpRequest.Builder builder = HttpRequest.newBuilder(URI.create(url)).timeout(Duration.ofSeconds(30)); headers.forEach(builder::header); builder.method(method, body == null ? HttpRequest.BodyPublishers.noBody() : HttpRequest.BodyPublishers.ofString(body)); HttpResponse<InputStream> response = httpClient.send(builder.build(), HttpResponse.BodyHandlers.ofInputStream()); Map<String, String> responseHeaders = response.headers().map().entrySet().stream().collect(java.util.stream.Collectors.toMap(Map.Entry::getKey, e -> e.getValue().isEmpty() ? "" : e.getValue().get(0), (left, right) -> left)); try (InputStream stream = response.body()) { byte[] data = stream.readNBytes(maxResponseBodyBytes + 1); if (data.length > maxResponseBodyBytes) throw new ResponseBodyLimitException(response.statusCode(), responseHeaders); return new RawResponse(response.statusCode(), responseHeaders, new String(data, StandardCharsets.UTF_8)); } }
}
