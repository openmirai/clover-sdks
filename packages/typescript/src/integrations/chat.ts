import { CloverClient, EmailAddress, SendEmailRequest } from "../index.js";

export interface CloverChatConfig {
  /** Envelope sender used for all outbound chat emails. */
  fromAddress: string;
  fromName?: string;
  apiKey: string;
  /** Explicit API endpoint unless an already-configured client is injected. */
  baseUrl?: string;
  webhookSecret?: string;
  client?: CloverClient;
  /** Application-owned signature verifier; required when a webhook secret is configured. */
  verifyWebhook?: (input: {
    rawBody: string;
    signature: string;
    secret: string;
  }) => boolean | Promise<boolean>;
}

export interface CloverChatMessage {
  text: string;
  subject?: string;
  idempotencyKey?: string;
  messageId?: string;
  inReplyTo?: string;
  references?: readonly string[];
}

export interface CloverChatInboundMessage {
  messageId?: string;
  threadId?: string;
  inReplyTo?: string;
  references: readonly string[];
  from?: EmailAddress;
  text: string;
  headers: Readonly<Record<string, string>>;
}

export interface CloverChatWebhook {
  rawBody: string;
  signature: string;
  headers: Readonly<Record<string, string>>;
  payload: {
    text: string;
    from?: EmailAddress;
    headers?: Readonly<Record<string, string>>;
  };
}

export class CloverChatUnsupportedOperationError extends Error {
  constructor(operation: "edit" | "delete" | "reaction" | "typing") {
    super(`Clover chat does not support ${operation}`);
    this.name = "CloverChatUnsupportedOperationError";
  }
}

export class CloverChatWebhookVerificationError extends Error {
  constructor() {
    super("Clover chat webhook signature verification failed");
    this.name = "CloverChatWebhookVerificationError";
  }
}

/**
 * Typed core for the optional Vercel Chat SDK integration. The `chat` package
 * remains an optional peer: consumers can map these methods to their Chat
 * adapter without adding a mandatory framework dependency to Clover SDK.
 */
export class CloverChatAdapter {
  private readonly client: CloverClient;
  private readonly from: EmailAddress;

  constructor(private readonly config: CloverChatConfig) {
    if (!config.fromAddress.trim()) throw new TypeError("fromAddress is required");
    if (!config.apiKey.trim()) throw new TypeError("apiKey is required");
    if (!config.client && !config.baseUrl?.trim())
      throw new TypeError("baseUrl or client is required");
    this.from = {
      address: config.fromAddress,
      ...(config.fromName ? { name: config.fromName } : {}),
    };
    this.client =
      config.client ?? new CloverClient({ baseUrl: config.baseUrl!, apiKey: config.apiKey });
    if (config.webhookSecret && !config.verifyWebhook)
      throw new TypeError("verifyWebhook is required when webhookSecret is configured");
  }

  async openDM(
    to: EmailAddress,
    subject = "Clover chat",
  ): Promise<{ id: string; threadId: string }> {
    const key = this.key("open", `${to.address}:${subject}`);
    const accepted = await this.client.send(this.email(to, { text: "", subject }, key), key);
    return { id: accepted.id, threadId: this.threadId(accepted.id) };
  }

  async post(
    to: EmailAddress,
    message: CloverChatMessage,
  ): Promise<{ id: string; threadId: string }> {
    const key =
      message.idempotencyKey ??
      this.key("post", message.messageId ?? `${message.inReplyTo ?? "root"}:${message.text}`);
    const accepted = await this.client.send(this.email(to, message, key), key);
    return { id: accepted.id, threadId: this.threadId(message.inReplyTo ?? accepted.id) };
  }

  async inbound(webhook: CloverChatWebhook): Promise<CloverChatInboundMessage> {
    if (this.config.webhookSecret) {
      const valid = await this.config.verifyWebhook!({
        rawBody: webhook.rawBody,
        signature: webhook.signature,
        secret: this.config.webhookSecret,
      });
      if (!valid) throw new CloverChatWebhookVerificationError();
    }
    const headers = webhook.payload.headers ?? webhook.headers;
    const messageId = headers["Message-ID"] ?? headers["message-id"];
    const inReplyTo = headers["In-Reply-To"] ?? headers["in-reply-to"];
    const references = (headers["References"] ?? headers["references"] ?? "")
      .split(/\s+/)
      .filter(Boolean);
    return {
      messageId,
      threadId: this.threadId(inReplyTo ?? references[0] ?? messageId),
      inReplyTo,
      references,
      from: webhook.payload.from,
      text: webhook.payload.text,
      headers,
    };
  }

  edit(): never {
    throw new CloverChatUnsupportedOperationError("edit");
  }
  delete(): never {
    throw new CloverChatUnsupportedOperationError("delete");
  }
  reaction(): never {
    throw new CloverChatUnsupportedOperationError("reaction");
  }
  typing(): never {
    throw new CloverChatUnsupportedOperationError("typing");
  }

  private email(
    to: EmailAddress,
    message: CloverChatMessage,
    messageKey: string,
  ): SendEmailRequest {
    const headers: Record<string, string> = {
      "Message-ID": message.messageId ?? `<clover-${this.token(messageKey)}@clover.local>`,
      ...(message.inReplyTo ? { "In-Reply-To": message.inReplyTo } : {}),
      ...(message.references?.length ? { References: message.references.join(" ") } : {}),
    };
    return {
      from: this.from,
      to: [to],
      subject: message.subject ?? "Clover chat",
      text: message.text,
      headers,
    };
  }

  private threadId(value: string | undefined): string {
    const normalized = value?.trim();
    if (!normalized) throw new Error("Clover chat response did not include a message id");
    return normalized;
  }
  private key(prefix: string, value: string): string {
    return `clover-chat-${prefix}-${this.token(value)}`;
  }
  private token(value: string): string {
    return (
      value
        .trim()
        .replace(/[^A-Za-z0-9_.-]+/g, "-")
        // Keep generated keys within Clover's 128-byte idempotency contract
        // after the `clover-chat-<operation>-` prefix is added.
        .slice(0, 96) || "message"
    );
  }
}
