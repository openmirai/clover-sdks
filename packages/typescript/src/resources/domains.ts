import { API_PREFIX, type Transport, withQuery } from "../transport.js";
import type {
  ConfigureDomainRequest,
  CreateDomainRequest,
  JsonObject,
  PaginatedData,
  RequestOptions,
} from "../types.js";

export interface ListDomainsOptions {
  page?: number;
  limit?: number;
  status?: string;
  provider?: string;
  sendingEnabled?: boolean;
  receivingEnabled?: boolean;
  [key: string]: string | number | boolean | undefined;
}

export class DomainsResource {
  constructor(private readonly transport: Transport) {}

  async create(
    request: CreateDomainRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(`${API_PREFIX}/domains`, {
      method: "POST",
      body: request,
      idempotencyKey: options.idempotencyKey,
    });
    return data;
  }

  async list(options: ListDomainsOptions = {}): Promise<PaginatedData<JsonObject>> {
    const query: Record<string, string | number | undefined> = {};
    for (const [key, value] of Object.entries(options)) {
      if (value !== undefined) query[key] = typeof value === "boolean" ? String(value) : value;
    }
    const { data } = await this.transport.request<PaginatedData<JsonObject>>(
      withQuery(`${API_PREFIX}/domains`, query),
      { method: "GET" },
    );
    return data;
  }

  async get(id: string): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(
      `${API_PREFIX}/domains/${encodeURIComponent(id)}`,
      { method: "GET" },
    );
    return data;
  }

  async update(
    id: string,
    request: ConfigureDomainRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(
      `${API_PREFIX}/domains/${encodeURIComponent(id)}`,
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
      `${API_PREFIX}/domains/${encodeURIComponent(id)}`,
      {
        method: "DELETE",
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async verify(
    id: string,
    options: RequestOptions & { idempotencyKey: string; force?: boolean },
  ): Promise<JsonObject> {
    const body = options.force === undefined ? {} : { force: options.force };
    const { data } = await this.transport.request<JsonObject>(
      `${API_PREFIX}/domains/${encodeURIComponent(id)}/verify`,
      {
        method: "POST",
        body,
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async dnsRecords(id: string): Promise<JsonObject> {
    const { data } = await this.transport.request<JsonObject>(
      `${API_PREFIX}/domains/${encodeURIComponent(id)}/dns-records`,
      { method: "GET" },
    );
    return data;
  }
}
