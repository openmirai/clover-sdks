import { platformEnvironmentPath, type Transport, withQuery } from "../transport.js";
import type {
  BatchEmailItem,
  EmailBatchAccepted,
  PlatformMessageAccepted,
  PlatformMessageDetail,
  PlatformMessagePage,
  PlatformMessageSummary,
  PlatformScope,
  PlatformSendMessageRequest,
  RequestOptions,
} from "../types.js";

const scopedMessagesPath = (scope: PlatformScope, suffix = ""): string =>
  platformEnvironmentPath(scope, `/messages${suffix}`);

export class PlatformMessagesResource {
  constructor(private readonly transport: Transport) {}

  async send(
    scope: PlatformScope,
    request: PlatformSendMessageRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<PlatformMessageAccepted> {
    const { data } = await this.transport.request<PlatformMessageAccepted>(
      scopedMessagesPath(scope),
      {
        method: "POST",
        body: request,
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async sendBatch(
    scope: PlatformScope,
    items: BatchEmailItem[],
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<EmailBatchAccepted> {
    const { data } = await this.transport.request<EmailBatchAccepted>(
      scopedMessagesPath(scope, "/batch"),
      {
        method: "POST",
        body: { items },
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async get(scope: PlatformScope, id: string): Promise<PlatformMessageDetail> {
    const { data } = await this.transport.request<PlatformMessageDetail>(
      scopedMessagesPath(scope, `/${encodeURIComponent(id)}`),
      { method: "GET" },
    );
    return data;
  }

  async list(
    scope: PlatformScope,
    options: Record<string, string | number | undefined> = {},
  ): Promise<PlatformMessagePage> {
    const { data } = await this.transport.request<PlatformMessagePage>(
      withQuery(scopedMessagesPath(scope), options),
      { method: "GET" },
    );
    return data;
  }

  async schedule(
    scope: PlatformScope,
    id: string,
    scheduledAt: string,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<PlatformMessageAccepted> {
    const { data } = await this.transport.request<PlatformMessageAccepted>(
      scopedMessagesPath(scope, `/${encodeURIComponent(id)}/schedule`),
      {
        method: "POST",
        body: { scheduledAt },
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async cancel(
    scope: PlatformScope,
    id: string,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<PlatformMessageSummary> {
    const { data } = await this.transport.request<PlatformMessageSummary>(
      scopedMessagesPath(scope, `/${encodeURIComponent(id)}/cancel`),
      { method: "POST", idempotencyKey: options.idempotencyKey },
    );
    return data;
  }
}
