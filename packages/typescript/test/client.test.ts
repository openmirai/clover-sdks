import { describe, expect, it } from "vitest";
import { CloverClient, CloverError } from "../src/index.js";
import type { BatchEmailItem } from "../src/index.js";
import { Resend, parseAddress, tagsToMap, toCloverSendRequest } from "../src/resend/index.js";

const email = {
  from: { address: "sender@example.com" },
  to: [{ address: "user@example.com" }],
  subject: "hello",
  text: "world",
};

function envelope<T>(data: T, status = 202): Response {
  return new Response(
    JSON.stringify({
      success: true,
      message: "ok",
      data,
      timestamp: "2026-08-20T00:00:00Z",
      requestId: "req_12345678",
    }),
    {
      status,
      headers: { "content-type": "application/json", "x-request-id": "req_12345678" },
    },
  );
}

describe("CloverClient V2", () => {
  it("posts /api/v1/emails, unwraps CommonResponse, and sends auth headers", async () => {
    const calls: Array<{ url: string; init: RequestInit }> = [];
    const client = new CloverClient({
      baseUrl: "https://api.example.test/",
      apiKey: "re_test_secret",
      fetch: async (url, init) => {
        calls.push({ url: String(url), init });
        return envelope({ id: "e1", status: "queued", request_id: "req_12345678", extra: "kept" });
      },
    });
    const result = await client.send(email, "idem-1234");
    expect(result.extra).toBe("kept");
    expect(result.id).toBe("e1");
    expect(calls[0].url).toBe("https://api.example.test/api/v1/emails");
    expect((calls[0].init.headers as Headers).get("authorization")).toBe("Bearer re_test_secret");
    expect((calls[0].init.headers as Headers).get("idempotency-key")).toBe("idem-1234");
  });

  it("retries GET and preserves ErrorResponse details", async () => {
    let calls = 0;
    let sleeps = 0;
    const client = new CloverClient({
      baseUrl: "https://api.example.test",
      apiKey: "k",
      maxRetries: 1,
      sleep: async () => {
        sleeps += 1;
      },
      fetch: async () => {
        calls += 1;
        return new Response(
          JSON.stringify({
            success: false,
            error: {
              code: 5000,
              type: "INTERNAL_ERROR",
              message: "Busy",
              details: "vendor",
            },
            requestId: "req_12345678",
          }),
          { status: 503 },
        );
      },
    });
    let thrown: unknown;
    try {
      await client.get("e1");
    } catch (error) {
      thrown = error;
    }
    expect(thrown).toBeInstanceOf(CloverError);
    expect((thrown as CloverError).error?.type).toBe("INTERNAL_ERROR");
    expect((thrown as CloverError).error?.details).toBe("vendor");
    expect(calls).toBe(2);
    expect(sleeps).toBe(1);
  });

  it("rejects invalid endpoint schemes and blank API keys", () => {
    expect(() => new CloverClient({ baseUrl: "ftp://api.example.test", apiKey: "secret" })).toThrow(
      TypeError,
    );
    expect(
      () => new CloverClient({ baseUrl: "https://user:pass@api.example.test", apiKey: "secret" }),
    ).toThrow(TypeError);
    expect(
      () => new CloverClient({ baseUrl: "https://api.example.test?token=leak", apiKey: "secret" }),
    ).toThrow(TypeError);
    expect(() => new CloverClient({ baseUrl: "https://api.example.test", apiKey: "  " })).toThrow(
      TypeError,
    );
  });

  it("enforces the canonical idempotency-key boundaries", async () => {
    const client = new CloverClient({
      baseUrl: "https://api.example.test",
      apiKey: "k",
      fetch: async () => envelope({}),
    });
    for (const key of [
      "a".repeat(7),
      "a".repeat(129),
      "a bad-key",
      "_" + "a".repeat(8),
      "a".repeat(8) + "\n",
    ]) {
      await expect(client.send(email, key)).rejects.toThrow(/idempotency key must match/);
    }
    await expect(client.send(email, "a".repeat(8))).resolves.toEqual({});
    await expect(client.send(email, "a".repeat(128))).resolves.toEqual({});
  });

  it("strips scheduled_at from batch items and uses /api/v1", async () => {
    let body = "";
    let url = "";
    const client = new CloverClient({
      baseUrl: "https://api.example.test",
      apiKey: "k",
      fetch: async (requestUrl, init) => {
        url = String(requestUrl);
        body = String(init?.body);
        return envelope({});
      },
    });
    const item = {
      subject: "hello",
      scheduled_at: "2030-01-01T00:00:00Z",
    } as unknown as BatchEmailItem;
    await client.sendBatch([item], "batch-1234");
    expect(url).toContain("/api/v1/emails/batch");
    expect(body).not.toContain("scheduled_at");
    expect(body).toContain('"items"');
  });

  it("exposes domains, apiKeys, and webhooks namespaces", async () => {
    const urls: string[] = [];
    const client = new CloverClient({
      baseUrl: "https://api.example.test",
      apiKey: "k",
      fetch: async (requestUrl) => {
        urls.push(String(requestUrl));
        return envelope({ items: [], pagination: { page: 1, limit: 50, total: 0 } }, 200);
      },
    });
    await client.domains.list();
    await client.apiKeys.list();
    await client.webhooks.list();
    expect(urls).toEqual([
      "https://api.example.test/api/v1/domains",
      "https://api.example.test/api/v1/api-keys",
      "https://api.example.test/api/v1/webhooks",
    ]);
  });

  it("rejects oversized responses before parsing", async () => {
    const client = new CloverClient({
      baseUrl: "https://api.example.test",
      apiKey: "k",
      maxResponseBodyBytes: 8,
      fetch: async () => new Response('{"data":"too long"}', { status: 200 }),
    });
    await expect(client.get("e1")).rejects.toMatchObject({
      status: 200,
      message: "Clover response body exceeds the configured limit",
    });
  });
});

describe("Resend façade", () => {
  it("maps string addresses and tags into Clover payloads", () => {
    expect(parseAddress("Ada <ada@example.com>")).toEqual({
      address: "ada@example.com",
      name: "Ada",
    });
    expect(tagsToMap([{ name: "campaign", value: "welcome" }])).toEqual({ campaign: "welcome" });
    const body = toCloverSendRequest({
      from: "Ada <ada@example.com>",
      to: ["bob@example.com"],
      subject: "Hi",
      html: "<p>Hi</p>",
      replyTo: "noreply@example.com",
      tags: [{ name: "campaign", value: "welcome" }],
      scheduledAt: "2030-01-01T00:00:00Z",
    });
    expect(body.from).toEqual({ address: "ada@example.com", name: "Ada" });
    expect(body.reply_to).toEqual([{ address: "noreply@example.com" }]);
    expect(body.tags).toEqual({ campaign: "welcome" });
    expect(body.scheduled_at).toBe("2030-01-01T00:00:00Z");
  });

  it("returns Result-style responses and auto-generates Idempotency-Key", async () => {
    const keys: string[] = [];
    const resend = new Resend("re_test", {
      baseUrl: "https://api.example.test",
      fetch: async (_url, init) => {
        const headers = new Headers(init?.headers);
        keys.push(headers.get("idempotency-key") ?? "");
        return envelope({ id: "e1", status: "queued", request_id: "req_1" });
      },
    });
    const result = await resend.emails.send({
      from: "sender@example.com",
      to: "user@example.com",
      subject: "hello",
      text: "world",
    });
    expect(result.error).toBeNull();
    expect(result.data).toEqual({ id: "e1" });
    expect(keys[0]).toMatch(/^[a-f0-9]{32}$/);
  });

  it("wraps batch arrays and maps schedule updates to POST /schedule", async () => {
    const calls: Array<{ url: string; body: string }> = [];
    const resend = new Resend("re_test", {
      baseUrl: "https://api.example.test",
      fetch: async (url, init) => {
        calls.push({ url: String(url), body: String(init?.body ?? "") });
        return envelope({
          id: "e1",
          status: "scheduled",
          request_id: "req_1",
          data: [{ id: "e1", status: "queued" }],
        });
      },
    });
    await resend.batch.send([
      { from: "a@example.com", to: "b@example.com", subject: "1", text: "1" },
    ]);
    await resend.emails.update({ id: "e1", scheduledAt: "2030-01-01T00:00:00Z" });
    expect(calls[0].url).toContain("/api/v1/emails/batch");
    expect(calls[0].body).toContain('"items"');
    expect(calls[1].url).toContain("/api/v1/emails/e1/schedule");
    expect(calls[1].body).toContain("scheduled_at");
  });

  it("maps API errors into Resend error objects without throwing", async () => {
    const resend = new Resend("re_test", {
      baseUrl: "https://api.example.test",
      fetch: async () =>
        new Response(
          JSON.stringify({
            success: false,
            error: { code: 4000, type: "VALIDATION_ERROR", message: "bad" },
          }),
          { status: 400 },
        ),
    });
    const result = await resend.emails.send({
      from: "a@example.com",
      to: "b@example.com",
      subject: "x",
      text: "y",
    });
    expect(result.data).toBeNull();
    expect(result.error).toMatchObject({
      message: "bad",
      statusCode: 400,
      name: "validation_error",
    });
  });
});
