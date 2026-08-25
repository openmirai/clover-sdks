import { CloverError } from "./errors.js";
import { APIKeysResource } from "./resources/api-keys.js";
import { DomainsResource } from "./resources/domains.js";
import { EmailsResource } from "./resources/emails.js";
import { WebhooksResource } from "./resources/webhooks.js";
import { PlatformMessagesResource } from "./resources/platform-messages.js";
import { createTransport, type Transport } from "./transport.js";
import type {
  BatchEmailItem,
  ClientOptions,
  EmailAccepted,
  EmailBatchAccepted,
  EmailPage,
  JsonObject,
  ListEmailsOptions,
  SendEmailRequest,
} from "./types.js";

/**
 * Native Clover V2 client (`/api/v1` + `CommonResponse` unwrap).
 *
 * Prefer namespaced resources (`emails`, `domains`, `apiKeys`, `webhooks`).
 * Top-level email helpers remain for callers that used the previous flat API.
 */
export class CloverClient {
  readonly emails: EmailsResource;
  readonly domains: DomainsResource;
  readonly apiKeys: APIKeysResource;
  readonly webhooks: WebhooksResource;
  readonly platformMessages: PlatformMessagesResource;
  private readonly transport: Transport;

  constructor(options: ClientOptions) {
    this.transport = createTransport(options);
    this.emails = new EmailsResource(this.transport);
    this.domains = new DomainsResource(this.transport);
    this.apiKeys = new APIKeysResource(this.transport);
    this.webhooks = new WebhooksResource(this.transport);
    this.platformMessages = new PlatformMessagesResource(this.transport);
  }

  /** @deprecated Prefer `emails.send` */
  async send(request: SendEmailRequest, idempotencyKey: string): Promise<EmailAccepted> {
    return this.emails.send(request, { idempotencyKey });
  }

  /** @deprecated Prefer `emails.sendBatch` */
  async sendBatch(
    items: BatchEmailItem[],
    idempotencyKey: string,
  ): Promise<EmailBatchAccepted | JsonObject> {
    return this.emails.sendBatch(items, { idempotencyKey });
  }

  /** @deprecated Prefer `emails.schedule` */
  async schedule(id: string, scheduledAt: string, idempotencyKey: string): Promise<EmailAccepted> {
    return this.emails.schedule(id, scheduledAt, { idempotencyKey });
  }

  /** @deprecated Prefer `emails.cancel` */
  async cancel(id: string, idempotencyKey: string): Promise<JsonObject> {
    return this.emails.cancel(id, { idempotencyKey });
  }

  /** @deprecated Prefer `emails.get` */
  async get(id: string): Promise<JsonObject> {
    return this.emails.get(id);
  }

  /** @deprecated Prefer `emails.list` */
  async list(options: ListEmailsOptions = {}): Promise<EmailPage> {
    return this.emails.list(options);
  }
}

export { CloverError };
