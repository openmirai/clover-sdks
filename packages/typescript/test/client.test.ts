import { describe, expect, it } from "vitest";
import {
  CloverClient,
  CloverError,
  type BatchEmailItem,
  type ClientOptions,
  type PlatformScope,
} from "../src/index.js";
import {
  Resend,
  mapToTags,
  parseAddress,
  tagsToMap,
  toCloverSendRequest,
} from "../src/resend/index.js";

const scope: PlatformScope = {
  accountId: "account/a",
  environmentId: "environment b",
};

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

function options(fetch: ClientOptions["fetch"]): ClientOptions {
  return {
    baseUrl: "https://api.example.test/",
    apiKey: "re_test_secret",
    accountId: scope.accountId,
    environmentId: scope.environmentId,
    fetch,
  };
}

function platformUrl(path: string): string {
  return `https://api.example.test${path}`;
}

describe("CloverClient V2 platform contract", () => {
  it("sends scoped camelCase messages and exposes explicit scope calls", async () => {
    let capturedUrl = "";
    let capturedBody = "";
    let capturedRequestId = "";
    const client = new CloverClient(
      options(async (url, init) => {
        capturedUrl = String(url);
        capturedBody = String(init?.body);
        capturedRequestId = new Headers(init?.headers).get("x-request-id") ?? "";
        return envelope({ id: "message-1", status: "accepted", requestId: "req_12345678" });
      }),
    );
    const accepted = await client.platformMessages.send(
      scope,
      {
        from: { address: "sender@example.com" },
        to: [{ address: "recipient@example.com" }],
        replyTo: [{ address: "reply@example.com" }],
        subject: "Hello",
        text: "Queued by Clover",
      },
      { idempotencyKey: "send-2026-0001" },
    );
    expect(capturedUrl).toBe(
      platformUrl("/api/v1/platform/accounts/account%2Fa/environments/environment%20b/messages"),
    );
    expect(JSON.parse(capturedBody)).toMatchObject({
      replyTo: [{ address: "reply@example.com" }],
    });
    expect(capturedBody).not.toContain("reply_to");
    expect(capturedRequestId).toMatch(/^req_[A-Za-z0-9_-]{8,128}$/);
    expect(accepted).toMatchObject({ id: "message-1", status: "accepted" });
  });

  it("uses scoped routes for all bound native resources", async () => {
    const calls: Array<{ method: string; url: string; body: string }> = [];
    const client = new CloverClient(
      options(async (url, init) => {
        const requestUrl = String(url);
        calls.push({
          method: init?.method ?? "GET",
          url: requestUrl,
          body: String(init?.body ?? ""),
        });
        if (requestUrl.endsWith("/domains")) {
          return envelope(
            {
              items: [{ id: "domain-1", name: "example.com", providerBindingId: "binding-1" }],
              pagination: { page: 1, limit: 20, total: 1 },
            },
            200,
          );
        }
        if (requestUrl.includes("/webhooks")) {
          return envelope({ items: [{ id: "hook-1", url: "https://example.com/hook" }] }, 200);
        }
        if (requestUrl.includes("/messages/batch")) {
          return envelope({ items: [{ id: "message-2", status: "accepted" }] });
        }
        if (requestUrl.includes("/messages")) {
          return envelope({ id: "message-1", status: "accepted" });
        }
        return envelope({ id: "resource-1", status: "ok" }, 200);
      }),
    );

    await client.emails.send(email, { idempotencyKey: "send-2026-0002" });
    await client.emails.sendBatch([{ ...email, scheduledAt: undefined } as BatchEmailItem], {
      idempotencyKey: "batch-2026-0002",
    });
    await client.emails.schedule("message/1", "2030-01-01T00:00:00Z", {
      idempotencyKey: "schedule-2026-0002",
    });
    await client.emails.cancel("message/1", { idempotencyKey: "cancel-2026-0002" });
    await client.emails.get("message/1");
    await client.emails.list({ page: 1, limit: 20, status: "accepted" });

    await client.domains.create(
      { domain: "example.com", providerBindingId: "binding-1" },
      { idempotencyKey: "domain-2026-0002" },
    );
    await client.domains.list({ page: 1, limit: 20 });
    await client.domains.verify("domain/1", { idempotencyKey: "verify-2026-0002" });

    await client.webhooks.create(
      { url: "https://example.com/hook", subscriptions: ["message.accepted"], enabled: true },
      { idempotencyKey: "hook-2026-0002" },
    );
    await client.webhooks.list({ cursor: "cursor-1", limit: 20, enabled: true });
    await client.webhooks.get("hook/1");
    await client.webhooks.update(
      "hook/1",
      { enabled: false },
      { idempotencyKey: "hook-2026-0003" },
    );
    await client.webhooks.remove("hook/1", { idempotencyKey: "hook-2026-0004" });
    await client.webhooks.rotateSecret("hook/1", {
      idempotencyKey: "hook-2026-0005",
      overlapSeconds: 60,
    });

    const paths = calls.map(
      ({ method, url }) => `${method} ${url.replace("https://api.example.test", "")}`,
    );
    expect(paths).toEqual([
      "POST /api/v1/platform/accounts/account%2Fa/environments/environment%20b/messages",
      "POST /api/v1/platform/accounts/account%2Fa/environments/environment%20b/messages/batch",
      "POST /api/v1/platform/accounts/account%2Fa/environments/environment%20b/messages/message%2F1/schedule",
      "POST /api/v1/platform/accounts/account%2Fa/environments/environment%20b/messages/message%2F1/cancel",
      "GET /api/v1/platform/accounts/account%2Fa/environments/environment%20b/messages/message%2F1",
      "GET /api/v1/platform/accounts/account%2Fa/environments/environment%20b/messages?page=1&limit=20&status=accepted",
      "POST /api/v1/platform/accounts/account%2Fa/environments/environment%20b/domains",
      "GET /api/v1/platform/accounts/account%2Fa/environments/environment%20b/domains?page=1&limit=20",
      "POST /api/v1/platform/accounts/account%2Fa/environments/environment%20b/domains/domain%2F1/verify",
      "POST /api/v1/platform/accounts/account%2Fa/environments/environment%20b/webhooks",
      "GET /api/v1/platform/accounts/account%2Fa/environments/environment%20b/webhooks?cursor=cursor-1&limit=20&enabled=true",
      "GET /api/v1/platform/accounts/account%2Fa/environments/environment%20b/webhooks/hook%2F1",
      "PATCH /api/v1/platform/accounts/account%2Fa/environments/environment%20b/webhooks/hook%2F1",
      "DELETE /api/v1/platform/accounts/account%2Fa/environments/environment%20b/webhooks/hook%2F1",
      "POST /api/v1/platform/accounts/account%2Fa/environments/environment%20b/webhooks/hook%2F1/rotate-secret",
    ]);
    expect(calls[14]?.body).toContain('"overlap_seconds":60');
    expect(
      calls.every(({ url }) => !url.match(/\/api\/v1\/(emails|domains|api-keys|webhooks)(\/|$)/)),
    ).toBe(true);
  });

  it("supports explicit platform message batch/list/lifecycle calls", async () => {
    const paths: string[] = [];
    const client = new CloverClient(
      options(async (url, init) => {
        paths.push(`${init?.method ?? "GET"} ${String(url)}`);
        return envelope(
          String(url).includes("batch")
            ? { items: [{ id: "m-1", status: "accepted" }] }
            : { items: [], pagination: { page: 1, limit: 20, total: 0 } },
          String(url).includes("batch") ? 202 : 200,
        );
      }),
    );
    await client.platformMessages.sendBatch(scope, [email], { idempotencyKey: "batch-2026-0003" });
    await client.platformMessages.list(scope, { page: 1 });
    expect(paths[0]).toContain("/messages/batch");
    expect(paths[1]).toContain("/messages?page=1");
  });

  it("retries GET, preserves V2 ErrorResponse details, and validates config", async () => {
    let calls = 0;
    let sleeps = 0;
    const client = new CloverClient({
      ...options(async () => {
        calls += 1;
        return new Response(
          JSON.stringify({
            success: false,
            error: { code: 5000, type: "INTERNAL_ERROR", message: "Busy", details: "vendor" },
            requestId: "req_12345678",
          }),
          { status: 503 },
        );
      }),
      maxRetries: 1,
      sleep: async () => {
        sleeps += 1;
      },
    });
    await expect(client.get("m-1")).rejects.toMatchObject({ status: 503 });
    expect(calls).toBe(2);
    expect(sleeps).toBe(1);
    try {
      await client.get("m-1");
    } catch (error) {
      expect(error).toBeInstanceOf(CloverError);
      expect((error as CloverError).error?.type).toBe("INTERNAL_ERROR");
      expect((error as CloverError).error?.details).toBe("vendor");
      expect((error as CloverError).meta.requestId).toBe("req_12345678");
    }
    expect(() => new CloverClient({ ...options(async () => envelope({})), accountId: "" })).toThrow(
      /accountId and environmentId are required/,
    );
    expect(
      () => new CloverClient({ ...options(async () => envelope({})), environmentId: "" }),
    ).toThrow(/accountId and environmentId are required/);
  });

  it("rejects unsafe URLs, keys, and oversized responses", async () => {
    expect(
      () => new CloverClient({ ...options(async () => envelope({})), baseUrl: "ftp://bad" }),
    ).toThrow(TypeError);
    expect(() => new CloverClient({ ...options(async () => envelope({})), apiKey: "  " })).toThrow(
      TypeError,
    );
    const client = new CloverClient({
      ...options(async () => new Response('{"data":"too long"}', { status: 200 })),
      maxResponseBodyBytes: 8,
    });
    await expect(client.get("m-1")).rejects.toMatchObject({
      status: 200,
      message: "Clover response body exceeds the configured limit",
    });
    const valid = new CloverClient({ ...options(async () => envelope({})) });
    for (const key of ["a".repeat(7), "a".repeat(129), "a bad-key", "_" + "a".repeat(8)]) {
      await expect(valid.send(email, key)).rejects.toThrow(/idempotency key must match/);
    }
    await expect(valid.send(email, "a".repeat(8))).resolves.toEqual({});
  });

  it("fails closed on malformed success responses and accepts explicit 204 responses", async () => {
    const flat = new CloverClient({
      ...options(async () => new Response(JSON.stringify({ id: "legacy-flat" }), { status: 200 })),
      maxRetries: 0,
    });
    await expect(flat.get("m-1")).rejects.toMatchObject({
      status: 200,
      message: "Clover success response is not a valid CommonResponse",
    });

    const malformed = new CloverClient({
      ...options(async () => new Response("not-json", { status: 200 })),
      maxRetries: 0,
    });
    await expect(malformed.get("m-1")).rejects.toMatchObject({
      status: 200,
      message: "Clover success response is not a valid CommonResponse",
    });

    const noContent = new CloverClient({
      ...options(async () => new Response(null, { status: 204 })),
      maxRetries: 0,
    });
    await expect(
      noContent.webhooks.remove("hook-1", { idempotencyKey: "hook-2026-2040" }),
    ).resolves.toEqual({});
  });
});

describe("Resend-compatible façade", () => {
  const resendOptions = {
    baseUrl: "https://api.example.test",
    accountId: scope.accountId,
    environmentId: scope.environmentId,
  };

  it("maps addresses, tags, camelCase fields, and scoped attachments", () => {
    expect(parseAddress("Ada <ada@example.com>")).toEqual({
      address: "ada@example.com",
      name: "Ada",
    });
    expect(parseAddress(" bob@example.com ")).toEqual({ address: "bob@example.com" });
    expect(tagsToMap([{ name: "campaign", value: "welcome" }])).toEqual({ campaign: "welcome" });
    expect(tagsToMap([])).toBeUndefined();
    expect(mapToTags({ campaign: "welcome" })).toEqual([{ name: "campaign", value: "welcome" }]);
    expect(mapToTags(null)).toBeNull();
    const body = toCloverSendRequest({
      from: "Ada <ada@example.com>",
      to: ["bob@example.com"],
      subject: "Hi",
      html: "<p>Hi</p>",
      replyTo: "noreply@example.com",
      tags: [{ name: "campaign", value: "welcome" }],
      scheduledAt: "2030-01-01T00:00:00Z",
      attachments: [
        {
          objectKey: "uploads/one",
          filename: "one.txt",
          contentId: "cid-1",
          sizeBytes: 4,
          sha256: "a".repeat(64),
        },
      ],
    });
    expect(body.from).toEqual({ address: "ada@example.com", name: "Ada" });
    expect(body.replyTo).toEqual([{ address: "noreply@example.com" }]);
    expect(body.tags).toEqual({ campaign: "welcome" });
    expect(body.scheduledAt).toBe("2030-01-01T00:00:00Z");
    expect(body.attachments?.[0]).toMatchObject({
      objectKey: "uploads/one",
      contentType: "application/octet-stream",
      disposition: "inline",
      contentId: "cid-1",
    });
    expect(body).not.toHaveProperty("reply_to");
    expect(body).not.toHaveProperty("scheduled_at");
  });

  it("uses scoped email, batch, domain, and webhook routes", async () => {
    const calls: Array<{ method: string; url: string; body: string }> = [];
    const resend = new Resend("re_test", {
      ...resendOptions,
      fetch: async (url, init) => {
        const requestUrl = String(url);
        calls.push({
          method: init?.method ?? "GET",
          url: requestUrl,
          body: String(init?.body ?? ""),
        });
        if (requestUrl.endsWith("/domains")) {
          return envelope({ items: [{ id: "domain-1" }] }, 200);
        }
        if (requestUrl.includes("/messages/batch")) {
          return envelope({ items: [{ id: "message-2" }] });
        }
        if (requestUrl.includes("/messages")) {
          return envelope({ id: "message-1", status: "accepted" });
        }
        if (requestUrl.endsWith("/webhooks") && init?.method === "POST") {
          return envelope({ endpoint: { id: "hook-1" }, secret: "secret" }, 201);
        }
        return envelope({ id: "hook-1" }, 200);
      },
    });
    const sent = await resend.emails.send({
      from: "a@example.com",
      to: "b@example.com",
      subject: "x",
    });
    expect(sent.error).toBeNull();
    await resend.emails.get("message-1");
    await resend.emails.list({ page: 1, limit: 10 });
    await resend.emails.update({ id: "message-1", scheduledAt: "2030-01-01T00:00:00Z" });
    await resend.emails.cancel("message-1");
    await resend.batch.send([{ from: "a@example.com", to: "b@example.com", subject: "batch" }]);
    await resend.domains.create({ name: "example.com", providerBindingId: "binding-1" });
    await resend.domains.list({ page: 1, limit: 10 });
    await resend.domains.verify("domain-1");
    const webhook = await resend.webhooks.create({
      endpoint: "https://example.com/hook",
      events: ["message.accepted"],
    });
    expect(webhook.data).toMatchObject({ id: "hook-1", signing_secret: "secret" });
    await resend.webhooks.get("hook-1");
    await resend.webhooks.list({ cursor: "cursor-1", limit: 10, enabled: true });
    await resend.webhooks.remove("hook-1");

    expect(calls[0]?.url).toContain(
      "/api/v1/platform/accounts/account%2Fa/environments/environment%20b/messages",
    );
    expect(
      calls.every(({ url }) => !url.match(/\/api\/v1\/(emails|domains|api-keys|webhooks)(\/|$)/)),
    ).toBe(true);
    expect(calls.find(({ url }) => url.includes("/messages/batch"))?.body).toContain('"items"');
    expect(calls.find(({ url }) => url.includes("/schedule"))?.body).toContain('"scheduledAt"');
  });

  it("returns Result-style V2 errors and requires a scope", async () => {
    expect(() => new Resend("re_test", { baseUrl: "https://api.example.test" })).toThrow(
      /accountId and environmentId are required/,
    );
    const resend = new Resend("re_test", {
      ...resendOptions,
      fetch: async () =>
        new Response(
          JSON.stringify({
            success: false,
            error: { code: 4000, type: "VALIDATION_ERROR", message: "bad" },
            requestId: "req_12345678",
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
    expect(result.headers).toEqual({ "x-request-id": "req_12345678" });
  });
});
