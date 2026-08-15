import { describe, expect, it } from "vitest";
import { CloverClient } from "../src/index.js";
import { CLOVER_CLIENT, CloverNestModule } from "../src/integrations/nestjs.js";
import {
  CloverChatAdapter,
  CloverChatUnsupportedOperationError,
  CloverChatWebhookVerificationError,
} from "../src/integrations/chat.js";

describe("optional framework adapters", () => {
  it("creates a Nest-compatible dynamic module without loading Nest", () => {
    const dynamic = CloverNestModule.forRoot({
      baseUrl: "https://api.example.test",
      apiKey: "secret",
    });
    expect(dynamic.exports).toEqual([CLOVER_CLIENT]);
    expect(dynamic.providers[0].useFactory()).toBeInstanceOf(CloverClient);
  });

  it("maps RFC threading headers and uses Clover outbound operations", async () => {
    const calls: RequestInit[] = [];
    const client = new CloverClient({
      baseUrl: "https://api.example.test",
      apiKey: "secret",
      fetch: async (_url, init) => {
        calls.push(init);
        return new Response(JSON.stringify({ id: "<reply@example.test>" }), { status: 202 });
      },
    });
    const adapter = new CloverChatAdapter({
      fromAddress: "bot@example.test",
      baseUrl: "https://api.example.test",
      apiKey: "secret",
      client,
    });
    const opened = await adapter.openDM({ address: "user@example.test" });
    expect(opened.threadId).toBe("<reply@example.test>");
    const result = await adapter.post(
      { address: "user@example.test" },
      { text: "hello", inReplyTo: "<root@example.test>", references: ["<root@example.test>"] },
    );
    expect(result.threadId).toBe("<root@example.test>");
    expect(JSON.parse(String(calls[1].body)).headers["In-Reply-To"]).toBe("<root@example.test>");
    const inbound = await adapter.inbound({
      rawBody: "{}",
      signature: "",
      headers: {},
      payload: { text: "reply", headers: { "Message-ID": "<m1>", "In-Reply-To": "<root>" } },
    });
    expect(inbound.threadId).toBe("<root>");
    expect(() => adapter.edit()).toThrow(CloverChatUnsupportedOperationError);
    expect(() => adapter.delete()).toThrow(CloverChatUnsupportedOperationError);
    expect(() => adapter.reaction()).toThrow(CloverChatUnsupportedOperationError);
    expect(() => adapter.typing()).toThrow(CloverChatUnsupportedOperationError);
  });

  it("fails closed when a webhook verifier rejects", async () => {
    const adapter = new CloverChatAdapter({
      fromAddress: "bot@example.test",
      baseUrl: "https://api.example.test",
      apiKey: "secret",
      webhookSecret: "hook",
      verifyWebhook: async () => false,
    });
    await expect(
      adapter.inbound({ rawBody: "{}", signature: "bad", headers: {}, payload: { text: "x" } }),
    ).rejects.toBeInstanceOf(CloverChatWebhookVerificationError);
  });

  it("bounds generated chat idempotency keys and keeps retries deterministic", async () => {
    const calls: RequestInit[] = [];
    const client = new CloverClient({
      baseUrl: "https://api.example.test",
      apiKey: "secret",
      fetch: async (_url, init) => {
        calls.push(init);
        return new Response(JSON.stringify({ id: "<reply@example.test>" }), { status: 202 });
      },
    });
    const adapter = new CloverChatAdapter({
      fromAddress: "bot@example.test",
      baseUrl: "https://api.example.test",
      apiKey: "secret",
      client,
    });
    const message = { text: "x".repeat(512) };
    await adapter.post({ address: "user@example.test" }, message);
    await adapter.post({ address: "user@example.test" }, message);
    const keys = calls.map((init) => new Headers(init.headers).get("Idempotency-Key"));
    expect(keys[0]).toBeTruthy();
    expect(keys[0]).toMatch(/^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$/);
    expect(keys[0]?.length).toBeLessThanOrEqual(128);
    expect(keys[0]).toBe(keys[1]);
  });
});
