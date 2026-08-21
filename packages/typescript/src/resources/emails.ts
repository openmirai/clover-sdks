import { API_PREFIX, type Transport, withQuery } from "../transport.js";
import type {
  BatchEmailItem,
  EmailAccepted,
  EmailBatchAccepted,
  EmailPage,
  JsonObject,
  ListEmailsOptions,
  RequestOptions,
  SendEmailRequest,
} from "../types.js";

export class EmailsResource {
  constructor(private readonly transport: Transport) {}

  async send(
    request: SendEmailRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<EmailAccepted> {
    const { data } = await this.transport.request<EmailAccepted>(`${API_PREFIX}/emails`, {
      method: "POST",
      body: request,
      idempotencyKey: options.idempotencyKey,
    });
    return data;
  }

  async sendBatch(
    items: BatchEmailItem[],
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<EmailBatchAccepted> {
    const sanitized = items.map(({ scheduled_at: _scheduledAt, ...item }) => item);
    const { data } = await this.transport.request<EmailBatchAccepted>(
      `${API_PREFIX}/emails/batch`,
      {
        method: "POST",
        body: { items: sanitized },
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async schedule(
    id: string,
    scheduledAt: string,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<EmailAccepted> {
    const { data } = await this.transport.request<EmailAccepted>(
      `${API_PREFIX}/emails/${encodeURIComponent(id)}/schedule`,
      {
        method: "POST",
        body: { scheduled_at: scheduledAt },
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async cancel(
    id: string,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(
      `${API_PREFIX}/emails/${encodeURIComponent(id)}/cancel`,
      {
        method: "POST",
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async get(id: string): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(
      `${API_PREFIX}/emails/${encodeURIComponent(id)}`,
      { method: "GET" },
    );
    return data;
  }

  async list(options: ListEmailsOptions = {}): Promise<EmailPage> {
    const { data } = await this.transport.request<EmailPage>(
      withQuery(`${API_PREFIX}/emails`, options),
      { method: "GET" },
    );
    return data;
  }
}
