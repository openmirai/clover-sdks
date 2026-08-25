import { API_PREFIX, type Transport, withQuery } from "../transport.js";
import type {
  PlatformMessageAccepted,
  PlatformMessagePage,
  PlatformScope,
  PlatformSendMessageRequest,
  RequestOptions,
} from "../types.js";

const scopedPath = (scope: PlatformScope, suffix = ""): string => {
  const accountId = scope.accountId.trim();
  const environmentId = scope.environmentId.trim();
  if (!accountId || !environmentId) {
    throw new TypeError("accountId and environmentId are required");
  }
  return `${API_PREFIX}/platform/accounts/${encodeURIComponent(accountId)}/environments/${encodeURIComponent(environmentId)}/messages${suffix}`;
};

export class PlatformMessagesResource {
  constructor(private readonly transport: Transport) {}

  async send(
    scope: PlatformScope,
    request: PlatformSendMessageRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<PlatformMessageAccepted> {
    const { data } = await this.transport.request<PlatformMessageAccepted>(scopedPath(scope), {
      method: "POST",
      body: request,
      idempotencyKey: options.idempotencyKey,
    });
    return data;
  }

  async get(scope: PlatformScope, id: string): Promise<PlatformMessageAccepted> {
    const { data } = await this.transport.request<PlatformMessageAccepted>(
      scopedPath(scope, `/${encodeURIComponent(id)}`),
      { method: "GET" },
    );
    return data;
  }

  async list(
    scope: PlatformScope,
    options: Record<string, string | number | undefined> = {},
  ): Promise<PlatformMessagePage> {
    const { data } = await this.transport.request<PlatformMessagePage>(
      withQuery(scopedPath(scope), options),
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
      scopedPath(scope, `/${encodeURIComponent(id)}/schedule`),
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
  ): Promise<PlatformMessageAccepted> {
    const { data } = await this.transport.request<PlatformMessageAccepted>(
      scopedPath(scope, `/${encodeURIComponent(id)}/cancel`),
      { method: "POST", idempotencyKey: options.idempotencyKey },
    );
    return data;
  }
}
