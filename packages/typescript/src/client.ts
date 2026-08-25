import { CloverError } from "./errors.js";
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
  PlatformMessageDetail,
  PlatformMessageSummary,
} from "./types.js";

/**
 * Native Clover V2 client over account/environment-scoped platform routes.
 *
 * The constructor binds `emails`, `domains`, and `webhooks` to one platform
 * account/environment. Platform API keys remain a dashboard-only control-plane
 * surface and are intentionally not exposed by this server-to-server client.
 */
export class CloverClient {
  readonly accountId: string;
  readonly environmentId: string;
  readonly emails: EmailsResource;
  readonly domains: DomainsResource;
  readonly webhooks: WebhooksResource;
  readonly platformMessages: PlatformMessagesResource;
  private readonly transport: Transport;

  constructor(options: ClientOptions) {
    this.accountId = options.accountId.trim();
    this.environmentId = options.environmentId.trim();
    if (!this.accountId || !this.environmentId) {
      throw new TypeError("accountId and environmentId are required");
    }
    this.transport = createTransport(options);
    const scope = { accountId: this.accountId, environmentId: this.environmentId };
    this.emails = new EmailsResource(this.transport, scope);
    this.domains = new DomainsResource(this.transport, scope);
    this.webhooks = new WebhooksResource(this.transport, scope);
    this.platformMessages = new PlatformMessagesResource(this.transport);
  }

  async send(request: SendEmailRequest, idempotencyKey: string): Promise<EmailAccepted> {
    return this.emails.send(request, { idempotencyKey });
  }

  async sendBatch(
    items: BatchEmailItem[],
    idempotencyKey: string,
  ): Promise<EmailBatchAccepted | JsonObject> {
    return this.emails.sendBatch(items, { idempotencyKey });
  }

  async schedule(id: string, scheduledAt: string, idempotencyKey: string): Promise<EmailAccepted> {
    return this.emails.schedule(id, scheduledAt, { idempotencyKey });
  }

  async cancel(id: string, idempotencyKey: string): Promise<PlatformMessageSummary> {
    return this.emails.cancel(id, { idempotencyKey });
  }

  async get(id: string): Promise<PlatformMessageDetail> {
    return this.emails.get(id);
  }

  async list(options: ListEmailsOptions = {}): Promise<EmailPage> {
    return this.emails.list(options);
  }
}

export { CloverError };
