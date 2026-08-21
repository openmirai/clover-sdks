import { API_PREFIX, type Transport, withQuery } from "../transport.js";
import type {
  CreateAPIKeyRequest,
  JsonObject,
  PaginatedData,
  RequestOptions,
  UpdateAPIKeyRequest,
} from "../types.js";

export interface ListAPIKeysOptions {
  page?: number;
  limit?: number;
  [key: string]: string | number | undefined;
}

export class APIKeysResource {
  constructor(private readonly transport: Transport) {}

  async create(
    request: CreateAPIKeyRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(`${API_PREFIX}/api-keys`, {
      method: "POST",
      body: request,
      idempotencyKey: options.idempotencyKey,
    });
    return data;
  }

  async list(options: ListAPIKeysOptions = {}): Promise<PaginatedData<JsonObject>> {
    const { data } = await this.transport.request<PaginatedData<JsonObject>>(
      withQuery(`${API_PREFIX}/api-keys`, options),
      { method: "GET" },
    );
    return data;
  }

  async update(
    id: string,
    request: UpdateAPIKeyRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(
      `${API_PREFIX}/api-keys/${encodeURIComponent(id)}`,
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
      `${API_PREFIX}/api-keys/${encodeURIComponent(id)}`,
      {
        method: "DELETE",
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }
}
