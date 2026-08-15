package dev.clover.sdk;

import java.util.*;
import java.util.concurrent.atomic.AtomicInteger;
import org.junit.jupiter.api.Test;

/** Dependency-free deterministic conformance test; run its main method. */
public final class ClientConformanceTest {
  @Test
  public void conformance() throws Exception {
    AtomicInteger calls = new AtomicInteger();
    CloverClient.Transport transport = (method, url, headers, body) -> {
      if (calls.incrementAndGet() == 1) return new CloverClient.RawResponse(503, Map.of(), "{\"type\":\"about:blank\",\"title\":\"Busy\",\"status\":503,\"code\":\"BUSY\",\"request_id\":\"req_12345678\",\"vendor\":{\"x\":1}}");
      return new CloverClient.RawResponse(202, Map.of("X-Request-ID", "req_12345678"), "{\"id\":\"e1\",\"status\":\"queued\",\"extra\":true}");
    };
    CloverClient client = new CloverClient("https://api.example.test", "secret", "test-agent", 1, 0, transport, ignored -> {});
    var httpClientField = CloverClient.class.getDeclaredField("httpClient");
    httpClientField.setAccessible(true);
    assert httpClientField.get(client) instanceof java.net.http.HttpClient;
    Map<String, Object> result = client.send(Map.of("subject", "hello"), "idem-1234");
    assert Boolean.TRUE.equals(result.get("extra"));
    assert calls.get() == 2;

    List<String> urls = new ArrayList<>();
    CloverClient encodedClient = new CloverClient("https://api.example.test", "secret", "test-agent", 0, 0, (method, url, headers, body) -> {
      urls.add(url);
      return new CloverClient.RawResponse(200, Map.of(), "{\"id\":\"e/1\",\"unicode\":\"\\u0E01\"}");
    }, ignored -> {});
    Map<String, Object> encoded = encodedClient.get("e/1 ?#");
    assert urls.get(0).equals("https://api.example.test/v1/emails/e%2F1%20%3F%23");
    assert "ก".equals(encoded.get("unicode"));

    boolean invalidJsonRejected = false;
    try { JsonCodec.parse("{\"ok\":true} trailing"); } catch (IllegalArgumentException expected) { invalidJsonRejected = true; }
    assert invalidJsonRejected;
    for (String malformed : List.of("{\"ok\":true,}", "{\"ok\":1}x", "{\"ok\":\"" + (char) 1 + "\"}", "{\"ok\":\"\\q\"}", "{\"ok\":\"\\uD800\"}", "01", "1.", "1e+")) {
      try { JsonCodec.parse(malformed); throw new AssertionError("accepted malformed JSON: " + malformed); }
      catch (IllegalArgumentException expected) { }
    }
    Object escaped = JsonCodec.parse("{\"text\":\"\\b\\f\\n\\r\\t\\uD834\\uDD1E\"}");
    assert "\b\f\n\r\t𝄞".equals(((Map<?, ?>) escaped).get("text"));
    String controls = JsonCodec.stringify("\u0000\u0001\u001f\b\f\n\r\t");
    assert "\"\\u0000\\u0001\\u001f\\b\\f\\n\\r\\t\"".equals(controls);
    boolean invalidStringRejected = false;
    try { JsonCodec.stringify("\uD800"); } catch (IllegalArgumentException expected) { invalidStringRejected = true; }
    assert invalidStringRejected;
    try {
      new CloverClient("https://api.example.test", "secret", "test-agent", 0, 0, (method, url, headers, body) -> new CloverClient.RawResponse(400, Map.of(), "{\"type\":\"about:blank\",\"title\":\"Bad request\"}"), ignored -> {}).get("e1");
      throw new AssertionError("RFC problem without code was accepted");
    } catch (CloverClient.CloverException expected) {
      assert expected.problem() != null;
      assert "Bad request".equals(expected.problem().get("title"));
    }
    try {
      new CloverClient("https://api.example.test", "secret", "test-agent", 0, 0, (method, url, headers, body) -> new CloverClient.RawResponse(502, Map.of("X-Request-ID", "req_bad"), "not-json"), ignored -> {}).get("e1");
      throw new AssertionError("malformed error body was accepted");
    } catch (CloverClient.CloverException expected) {
      assert expected.status() == 502;
      assert expected.problem() != null;
      assert "not-json".equals(expected.problem().get("raw"));
      assert "req_bad".equals(expected.metadata().requestId());
    }
    AtomicInteger retryDelay = new AtomicInteger();
    try {
      new CloverClient("https://api.example.test", "secret", "test-agent", 1, 100, (method, url, headers, body) -> new CloverClient.RawResponse(503, Map.of("Retry-After", "-1"), "{\"type\":\"about:blank\",\"title\":\"Busy\"}"), delay -> { retryDelay.set((int) delay); }).get("e1");
      throw new AssertionError("retry response was accepted");
    } catch (CloverClient.CloverException expected) {
      assert retryDelay.get() == 100;
    }
    try {
      new CloverClient("https://api.example.test", "secret", "test-agent", 0, 0, (method, url, headers, body) -> new CloverClient.RawResponse(200, Map.of(), "{\"data\":\"too long\"}"), ignored -> {}, 8).get("e1");
      throw new AssertionError("oversized response was accepted");
    } catch (CloverClient.CloverException expected) {
      assert expected.status() == 200;
      assert expected.getMessage().contains("exceeds the configured limit");
    }
    for (String[] invalid : List.of(new String[]{"", "secret"}, new String[]{"ftp://api.example.test", "secret"}, new String[]{"https://", "secret"}, new String[]{"https://user:pass@api.example.test", "secret"}, new String[]{"https://api.example.test?token=leak", "secret"}, new String[]{"https://api.example.test", "  "})) {
      boolean rejected = false;
      try { new CloverClient(invalid[0], invalid[1]); } catch (IllegalArgumentException expected) { rejected = true; }
      assert rejected : "accepted invalid configuration";
    }
    for (String invalidKey : List.of("a".repeat(7), "a".repeat(129), "a bad-key", "_" + "a".repeat(8), "a".repeat(8) + "\n", "a".repeat(8) + "\r", "a".repeat(8) + "\0")) {
      boolean rejected = false;
      try { client.send(Map.of("subject", "hello"), invalidKey); } catch (IllegalArgumentException expected) { rejected = true; }
      assert rejected : "accepted invalid idempotency key";
    }
    CloverClient keyClient = new CloverClient("https://api.example.test", "secret", "test-agent", 0, 0, (method, url, headers, body) -> new CloverClient.RawResponse(202, Map.of(), "{}"), ignored -> {});
    keyClient.send(Map.of("subject", "hello"), "a".repeat(8));
    keyClient.send(Map.of("subject", "hello"), "a".repeat(128));
    System.out.println("ok");
  }

  public static void main(String[] args) throws Exception {
    new ClientConformanceTest().conformance();
  }
}
