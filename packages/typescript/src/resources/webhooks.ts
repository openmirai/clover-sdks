import { API_PREFIX, type Transport, withQuery } from "../transport.js";
import type {
  CreateWebhookRequest,
  JsonObject,
  PaginatedData,
  RequestOptions,
  UpdateWebhookRequest,
} from "../types.js";

export interface ListWebhooksOptions {
  page?: number;
  limit?: number;
  [key: string]: string | number | undefined;
}

export class WebhooksResource {
  constructor(private readonly transport: Transport) {}

  async create(
    request: CreateWebhookRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(`${API_PREFIX}/webhooks`, {
      method: "POST",
      body: request,
      idempotencyKey: options.idempotencyKey,
    });
    return data;
  }

  async list(options: ListWebhooksOptions = {}): Promise<PaginatedData<JsonObject>> {
    const { data } = await this.transport.request<PaginatedData<JsonObject>>(
      withQuery(`${API_PREFIX}/webhooks`, options),
      { method: "GET" },
    );
    return data;
  }

  async get(id: string): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(
      `${API_PREFIX}/webhooks/${encodeURIComponent(id)}`,
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
      `${API_PREFIX}/webhooks/${encodeURIComponent(id)}`,
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
      `${API_PREFIX}/webhooks/${encodeURIComponent(id)}`,
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
      `${API_PREFIX}/webhooks/${encodeURIComponent(id)}/rotate-secret`,
      {
        method: "POST",
        body,
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }
}
