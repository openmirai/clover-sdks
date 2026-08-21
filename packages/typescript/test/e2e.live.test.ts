import { describe, expect, it } from "vitest";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { CloverClient } from "../src/index.js";
import { Resend } from "../src/resend/index.js";

const enabled = process.env.CLOVER_LIVE_E2E === "1";
const __dirname = path.dirname(fileURLToPath(import.meta.url));
const fixturePath =
  process.env.CLOVER_E2E_FIXTURE || path.join(__dirname, "../.live-e2e-fixture.json");

const suite = enabled ? describe : describe.skip;

suite("live Clover E2E", () => {
  function loadFixture() {
    if (!fs.existsSync(fixturePath)) {
      throw new Error(
        `missing fixture at ${fixturePath}; run scripts/live-e2e-bootstrap.mjs first`,
      );
    }
    return JSON.parse(fs.readFileSync(fixturePath, "utf8")) as {
      apiUrl: string;
      apiKey: string;
      domainId: string;
      domainName: string;
      from: string;
      mailpitUrl: string;
    };
  }

  async function waitForMailpit(
    mailpitUrl: string,
    subject: string,
    timeoutMs = 30_000,
  ): Promise<unknown> {
    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
      const response = await fetch(
        `${mailpitUrl}/api/v1/search?query=${encodeURIComponent(`subject:${subject}`)}`,
      );
      const body = (await response.json()) as { messages?: Array<{ Subject?: string }> };
      const hit = body.messages?.find((message) => message.Subject === subject);
      if (hit) return hit;
      await new Promise((resolve) => setTimeout(resolve, 500));
    }
    throw new Error(`Mailpit did not receive subject ${subject}`);
  }

  it("native client: send → Mailpit, idempotent replay, batch, schedule/cancel, domains list", async () => {
    const fixture = loadFixture();
    const client = new CloverClient({ baseUrl: fixture.apiUrl, apiKey: fixture.apiKey });
    const subject = `native-${Date.now()}`;
    const idem = `native-send-${Date.now()}`;

    const accepted = await client.emails.send(
      {
        from: { address: fixture.from },
        to: [{ address: "user@example.test" }],
        subject,
        text: "hello from native sdk",
      },
      { idempotencyKey: idem },
    );
    expect(accepted.id).toBeTruthy();

    const replay = await client.emails.send(
      {
        from: { address: fixture.from },
        to: [{ address: "user@example.test" }],
        subject,
        text: "hello from native sdk",
      },
      { idempotencyKey: idem },
    );
    expect(replay.id).toBe(accepted.id);

    await waitForMailpit(fixture.mailpitUrl, subject);

    const batchSubject = `batch-${Date.now()}`;
    const batch = await client.emails.sendBatch(
      [
        {
          from: { address: fixture.from },
          to: [{ address: "batch-a@example.test" }],
          subject: `${batchSubject}-a`,
          text: "a",
        },
        {
          from: { address: fixture.from },
          to: [{ address: "batch-b@example.test" }],
          subject: `${batchSubject}-b`,
          text: "b",
        },
      ],
      { idempotencyKey: `native-batch-${Date.now()}` },
    );
    expect(batch.data?.length).toBe(2);

    const scheduled = await client.emails.send(
      {
        from: { address: fixture.from },
        to: [{ address: "sched@example.test" }],
        subject: `sched-${Date.now()}`,
        text: "later",
        scheduled_at: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
      },
      { idempotencyKey: `native-sched-${Date.now()}` },
    );
    const cancelled = await client.emails.cancel(String(scheduled.id), {
      idempotencyKey: `native-cancel-${Date.now()}`,
    });
    expect(cancelled).toBeTruthy();

    const domains = await client.domains.list();
    expect(domains.items.some((item) => item.id === fixture.domainId)).toBe(true);
  }, 60_000);

  it("Resend façade: emails.send style against local Clover", async () => {
    const fixture = loadFixture();
    const resend = new Resend(fixture.apiKey, { baseUrl: fixture.apiUrl });
    const subject = `resend-${Date.now()}`;
    const result = await resend.emails.send({
      from: fixture.from,
      to: "user@example.test",
      subject,
      text: "hello from resend façade",
      tags: [{ name: "suite", value: "live" }],
    });
    expect(result.error).toBeNull();
    expect(result.data?.id).toBeTruthy();
    await waitForMailpit(fixture.mailpitUrl, subject);

    const listed = await resend.domains.list();
    expect(listed.error).toBeNull();
    expect(listed.data?.data.some((item) => item.id === fixture.domainId)).toBe(true);
  }, 60_000);
});
