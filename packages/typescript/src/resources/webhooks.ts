import { platformEnvironmentPath, type Transport, withQuery } from "../transport.js";
import type {
  CreateWebhookRequest,
  JsonObject,
  PlatformScope,
  RequestOptions,
  UpdateWebhookRequest,
  WebhookListPayload,
} from "../types.js";

export interface ListWebhooksOptions {
  cursor?: string;
  limit?: number;
  enabled?: boolean;
  [key: string]: string | number | boolean | undefined;
}

const webhooksPath = (scope: PlatformScope, suffix = ""): string =>
  platformEnvironmentPath(scope, `/webhooks${suffix}`);

export class WebhooksResource {
  constructor(
    private readonly transport: Transport,
    private readonly scope: PlatformScope,
  ) {}

  async create(
    request: CreateWebhookRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(webhooksPath(this.scope), {
      method: "POST",
      body: request,
      idempotencyKey: options.idempotencyKey,
    });
    return data;
  }

  async list(options: ListWebhooksOptions = {}): Promise<WebhookListPayload> {
    const { data } = await this.transport.request<WebhookListPayload>(
      withQuery(webhooksPath(this.scope), options),
      { method: "GET" },
    );
    return data;
  }

  async get(id: string): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(
      webhooksPath(this.scope, `/${encodeURIComponent(id)}`),
      { method: "GET" },
    );
    return data;
  }

  async update(
    id: string,
    request: UpdateWebhookRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(
      webhooksPath(this.scope, `/${encodeURIComponent(id)}`),
      {
        method: "PATCH",
        body: request,
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async remove(
    id: string,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(
      webhooksPath(this.scope, `/${encodeURIComponent(id)}`),
      {
        method: "DELETE",
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async rotateSecret(
    id: string,
    options: RequestOptions & { idempotencyKey: string; overlapSeconds?: number },
  ): Promise<JsonObject> {
    const body =
      options.overlapSeconds === undefined ? {} : { overlap_seconds: options.overlapSeconds };
    const { data } = await this.transport.request<JsonObject>(
      webhooksPath(this.scope, `/${encodeURIComponent(id)}/rotate-secret`),
      {
        method: "POST",
        body,
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }
}
