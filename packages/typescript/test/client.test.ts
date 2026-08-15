import { describe, expect, it } from "vitest";
import { CloverClient, CloverError } from "../src/index.js";
import type { BatchEmailItem } from "../src/index.js";

const email = {
  from: { address: "sender@example.com" },
  to: [{ address: "user@example.com" }],
  subject: "hello",
  text: "world",
};

describe("CloverClient", () => {
  it("sends bearer/idempotency headers and decodes accepted response", async () => {
    const calls: Array<{ url: string; init: RequestInit }> = [];
    const client = new CloverClient({
      baseUrl: "https://api.example.test/",
      apiKey: "re_test_secret",
      fetch: async (url, init) => {
        calls.push({ url: String(url), init });
        return new Response(
          JSON.stringify({ id: "e1", status: "queued", request_id: "req_12345678", extra: "kept" }),
          {
            status: 202,
            headers: { "content-type": "application/json", "x-request-id": "req_12345678" },
          },
        );
      },
    });
    const result = await client.send(email, "idem-1234");
    expect(result.extra).toBe("kept");
    expect((calls[0].init.headers as Headers).get("authorization")).toBe("Bearer re_test_secret");
    expect((calls[0].init.headers as Headers).get("idempotency-key")).toBe("idem-1234");
  });

  it("retries GET at most the configured bound and preserves problem fields", async () => {
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
            type: "about:blank",
            title: "Busy",
            status: 503,
            code: "BUSY",
            request_id: "req_12345678",
            vendor: { x: 1 },
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
    expect((thrown as CloverError).problem?.vendor).toEqual({ x: 1 });
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
      fetch: async () => new Response("{}", { status: 202 }),
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

  it("strips scheduled_at from batch items", async () => {
    let body = "";
    const client = new CloverClient({
      baseUrl: "https://api.example.test",
      apiKey: "k",
      fetch: async (_url, init) => {
        body = String(init.body);
        return new Response("{}", { status: 202 });
      },
    });
    const item = {
      subject: "hello",
      scheduled_at: "2030-01-01T00:00:00Z",
    } as unknown as BatchEmailItem;
    await client.sendBatch([item], "batch-1234");
    expect(body).not.toContain("scheduled_at");
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
