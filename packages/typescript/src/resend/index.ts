import { CloverError } from "../errors.js";
import {
  createIdempotencyKey,
  createTransport,
  type Transport,
  API_PREFIX,
  withQuery,
} from "../transport.js";
import type { ClientOptions, JsonObject, SendEmailRequest } from "../types.js";

export type ResendResult<T> =
  | { data: T; error: null; headers: Record<string, string> | null }
  | {
      data: null;
      error: { message: string; statusCode: number | null; name: string };
      headers: Record<string, string> | null;
    };

export interface ResendOptions {
  baseUrl?: string;
  userAgent?: string;
  fetch?: typeof globalThis.fetch;
  maxRetries?: number;
  retryBaseDelayMs?: number;
  maxResponseBodyBytes?: number;
  sleep?: (milliseconds: number) => Promise<void>;
}

export type Tag = { name: string; value: string };

export interface CreateEmailOptions {
  from: string;
  to: string | string[];
  subject: string;
  html?: string;
  text?: string;
  cc?: string | string[];
  bcc?: string | string[];
  replyTo?: string | string[];
  reply_to?: string | string[];
  headers?: Record<string, string>;
  tags?: Tag[];
  attachments?: Array<{
    filename?: string;
    content?: string;
    contentType?: string;
    content_type?: string;
    contentId?: string;
    content_id?: string;
  }>;
  scheduledAt?: string;
  scheduled_at?: string;
}

export interface IdempotentOptions {
  idempotencyKey?: string;
}

const ADDRESS_WITH_NAME = /^(.*?)\s*<([^>]+)>\s*$/;

export function parseAddress(value: string): { address: string; name?: string } {
  const trimmed = value.trim();
  const match = ADDRESS_WITH_NAME.exec(trimmed);
  if (!match) return { address: trimmed };
  const name = match[1].trim().replace(/^"|"$/g, "");
  const address = match[2].trim();
  return name ? { address, name } : { address };
}

function asAddressList(
  value: string | string[] | undefined,
): Array<{ address: string; name?: string }> | undefined {
  if (value === undefined) return undefined;
  const list = Array.isArray(value) ? value : [value];
  return list.map(parseAddress);
}

export function tagsToMap(tags: Tag[] | undefined): Record<string, string> | undefined {
  if (!tags?.length) return undefined;
  const map: Record<string, string> = {};
  for (const tag of tags) map[tag.name] = tag.value;
  return map;
}

export function mapToTags(tags: Record<string, string> | undefined | null): Tag[] | null {
  if (!tags) return null;
  return Object.entries(tags).map(([name, value]) => ({ name, value }));
}

export function toCloverSendRequest(payload: CreateEmailOptions): SendEmailRequest {
  const replyTo = payload.replyTo ?? payload.reply_to;
  const scheduledAt = payload.scheduledAt ?? payload.scheduled_at;
  const request: SendEmailRequest = {
    from: parseAddress(payload.from),
    to: asAddressList(payload.to) ?? [],
    subject: payload.subject,
  };
  const cc = asAddressList(payload.cc);
  const bcc = asAddressList(payload.bcc);
  const reply = asAddressList(replyTo);
  if (cc) request.cc = cc;
  if (bcc) request.bcc = bcc;
  if (reply) request.reply_to = reply;
  if (payload.html !== undefined) request.html = payload.html;
  if (payload.text !== undefined) request.text = payload.text;
  if (payload.headers) request.headers = payload.headers;
  const tags = tagsToMap(payload.tags);
  if (tags) request.tags = tags;
  if (scheduledAt) request.scheduled_at = scheduledAt;
  if (payload.attachments?.length) {
    request.attachments = payload.attachments.map((attachment) => ({
      filename: attachment.filename ?? "attachment",
      content_type: attachment.contentType ?? attachment.content_type ?? "application/octet-stream",
      disposition: attachment.contentId || attachment.content_id ? "inline" : "attachment",
      content_id: attachment.contentId ?? attachment.content_id ?? null,
      content: attachment.content,
    }));
  }
  return request;
}

function headersFromMeta(meta: {
  requestId?: string;
  replayed: boolean;
}): Record<string, string> | null {
  const headers: Record<string, string> = {};
  if (meta.requestId) headers["x-request-id"] = meta.requestId;
  if (meta.replayed) headers["idempotency-replayed"] = "true";
  return Object.keys(headers).length ? headers : null;
}

function mapErrorName(type: string | undefined, status: number | null): string {
  const normalized = (type ?? "").toLowerCase();
  if (normalized.includes("validation")) return "validation_error";
  if (normalized.includes("not_found") || status === 404) return "not_found";
  if (normalized.includes("unauthorized") || status === 401) return "invalid_api_key";
  if (normalized.includes("forbidden") || status === 403) return "restricted_api_key";
  if (normalized.includes("conflict") || status === 409) return "invalid_idempotency_key";
  if (normalized.includes("rate")) return "rate_limit_exceeded";
  return "application_error";
}

async function asResult<T>(
  run: () => Promise<{ data: T; meta: { requestId?: string; replayed: boolean } }>,
): Promise<ResendResult<T>> {
  try {
    const { data, meta } = await run();
    return { data, error: null, headers: headersFromMeta(meta) };
  } catch (error) {
    if (error instanceof CloverError) {
      return {
        data: null,
        error: {
          message: error.message,
          statusCode: error.status,
          name: mapErrorName(error.error?.type, error.status),
        },
        headers: headersFromMeta(error.meta),
      };
    }
    throw error;
  }
}

function resolveIdempotencyKey(options?: IdempotentOptions): string {
  return options?.idempotencyKey?.trim() || createIdempotencyKey();
}

class ResendEmails {
  constructor(private readonly transport: Transport) {}

  send(
    payload: CreateEmailOptions,
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ id: string }>> {
    return this.create(payload, options);
  }

  create(
    payload: CreateEmailOptions,
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ id: string }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{ id: string }>(`${API_PREFIX}/emails`, {
        method: "POST",
        body: toCloverSendRequest(payload),
        idempotencyKey: resolveIdempotencyKey(options),
      });
      return { data: { id: String(data.id) }, meta };
    });
  }

  get(id: string): Promise<ResendResult<JsonObject>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<JsonObject>(
        `${API_PREFIX}/emails/${encodeURIComponent(id)}`,
        { method: "GET" },
      );
      return { data, meta };
    });
  }

  list(
    options: { page?: number; limit?: number; cursor?: string; status?: string } = {},
  ): Promise<ResendResult<{ object: "list"; data: JsonObject[]; has_more: boolean }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{
        items?: JsonObject[];
        pagination?: { hasNext?: boolean };
      }>(withQuery(`${API_PREFIX}/emails`, options), { method: "GET" });
      const items = data.items ?? [];
      return {
        data: {
          object: "list",
          data: items,
          has_more: Boolean(data.pagination?.hasNext),
        },
        meta,
      };
    });
  }

  update(
    payload: { id: string; scheduledAt: string },
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ id: string; object: "email" }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{ id: string }>(
        `${API_PREFIX}/emails/${encodeURIComponent(payload.id)}/schedule`,
        {
          method: "POST",
          body: { scheduled_at: payload.scheduledAt },
          idempotencyKey: resolveIdempotencyKey(options),
        },
      );
      return { data: { id: String(data.id), object: "email" }, meta };
    });
  }

  cancel(
    id: string,
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ id: string; object: "email" }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{ id?: string }>(
        `${API_PREFIX}/emails/${encodeURIComponent(id)}/cancel`,
        {
          method: "POST",
          idempotencyKey: resolveIdempotencyKey(options),
        },
      );
      return { data: { id: String(data.id ?? id), object: "email" }, meta };
    });
  }
}

class ResendBatch {
  constructor(private readonly transport: Transport) {}

  send(
    payload: CreateEmailOptions[],
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ data: Array<{ id: string }> }>> {
    return this.create(payload, options);
  }

  create(
    payload: CreateEmailOptions[],
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ data: Array<{ id: string }> }>> {
    return asResult(async () => {
      const items = payload.map((item) => {
        const body = toCloverSendRequest(item);
        const { scheduled_at: _scheduledAt, ...rest } = body;
        return rest;
      });
      const { data, meta } = await this.transport.request<{ data?: Array<{ id: string }> }>(
        `${API_PREFIX}/emails/batch`,
        {
          method: "POST",
          body: { items },
          idempotencyKey: resolveIdempotencyKey(options),
        },
      );
      return {
        data: { data: (data.data ?? []).map((item) => ({ id: String(item.id) })) },
        meta,
      };
    });
  }
}

class ResendDomains {
  constructor(private readonly transport: Transport) {}

  create(
    payload: {
      name: string;
      region?: string;
      provider?: string;
      receivingEnabled?: boolean;
      tlsPolicy?: string;
    },
    options?: IdempotentOptions,
  ): Promise<ResendResult<JsonObject>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<JsonObject>(`${API_PREFIX}/domains`, {
        method: "POST",
        body: {
          name: payload.name,
          provider: payload.provider ?? "smtp",
          region: payload.region ?? "local",
          receivingEnabled: payload.receivingEnabled ?? false,
          tlsPolicy: payload.tlsPolicy ?? "opportunistic",
        },
        idempotencyKey: resolveIdempotencyKey(options),
      });
      return { data, meta };
    });
  }

  list(
    options: { page?: number; limit?: number } = {},
  ): Promise<ResendResult<{ object: "list"; data: JsonObject[] }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{ items?: JsonObject[] }>(
        withQuery(`${API_PREFIX}/domains`, options),
        { method: "GET" },
      );
      return { data: { object: "list", data: data.items ?? [] }, meta };
    });
  }

  get(id: string): Promise<ResendResult<JsonObject>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<JsonObject>(
        `${API_PREFIX}/domains/${encodeURIComponent(id)}`,
        { method: "GET" },
      );
      return { data, meta };
    });
  }

  remove(
    id: string,
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ id: string; object: "domain"; deleted: true }>> {
    return asResult(async () => {
      await this.transport.request(`${API_PREFIX}/domains/${encodeURIComponent(id)}`, {
        method: "DELETE",
        idempotencyKey: resolveIdempotencyKey(options),
      });
      return { data: { id, object: "domain", deleted: true as const }, meta: { replayed: false } };
    });
  }

  verify(
    id: string,
    options?: IdempotentOptions & { force?: boolean },
  ): Promise<ResendResult<JsonObject>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<JsonObject>(
        `${API_PREFIX}/domains/${encodeURIComponent(id)}/verify`,
        {
          method: "POST",
          body: options?.force === undefined ? {} : { force: options.force },
          idempotencyKey: resolveIdempotencyKey(options),
        },
      );
      return { data, meta };
    });
  }
}

class ResendApiKeys {
  constructor(private readonly transport: Transport) {}

  create(
    payload: { name: string; permission?: "full_access" | "sending_access"; domain_id?: string },
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ id: string; token: string }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{
        key?: { id?: string };
        token?: string;
      }>(`${API_PREFIX}/api-keys`, {
        method: "POST",
        body: payload,
        idempotencyKey: resolveIdempotencyKey(options),
      });
      return {
        data: { id: String(data.key?.id ?? ""), token: String(data.token ?? "") },
        meta,
      };
    });
  }

  list(
    options: { page?: number; limit?: number } = {},
  ): Promise<ResendResult<{ object: "list"; data: JsonObject[] }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{ items?: JsonObject[] }>(
        withQuery(`${API_PREFIX}/api-keys`, options),
        { method: "GET" },
      );
      return { data: { object: "list", data: data.items ?? [] }, meta };
    });
  }

  remove(
    id: string,
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ id: string; object: "api_key"; deleted: true }>> {
    return asResult(async () => {
      await this.transport.request(`${API_PREFIX}/api-keys/${encodeURIComponent(id)}`, {
        method: "DELETE",
        idempotencyKey: resolveIdempotencyKey(options),
      });
      return { data: { id, object: "api_key", deleted: true as const }, meta: { replayed: false } };
    });
  }
}

class ResendWebhooks {
  constructor(private readonly transport: Transport) {}

  create(
    payload: { endpoint: string; events: string[] },
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ id: string; signing_secret: string; object: "webhook" }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{ id?: string; secret?: string }>(
        `${API_PREFIX}/webhooks`,
        {
          method: "POST",
          body: {
            url: payload.endpoint,
            subscriptions: payload.events,
          },
          idempotencyKey: resolveIdempotencyKey(options),
        },
      );
      return {
        data: {
          object: "webhook",
          id: String(data.id ?? ""),
          signing_secret: String(data.secret ?? ""),
        },
        meta,
      };
    });
  }

  get(id: string): Promise<ResendResult<JsonObject>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<JsonObject>(
        `${API_PREFIX}/webhooks/${encodeURIComponent(id)}`,
        { method: "GET" },
      );
      return { data, meta };
    });
  }

  list(
    options: { page?: number; limit?: number } = {},
  ): Promise<ResendResult<{ object: "list"; data: JsonObject[] }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{ items?: JsonObject[] }>(
        withQuery(`${API_PREFIX}/webhooks`, options),
        { method: "GET" },
      );
      return { data: { object: "list", data: data.items ?? [] }, meta };
    });
  }

  remove(
    id: string,
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ id: string; object: "webhook"; deleted: true }>> {
    return asResult(async () => {
      await this.transport.request(`${API_PREFIX}/webhooks/${encodeURIComponent(id)}`, {
        method: "DELETE",
        idempotencyKey: resolveIdempotencyKey(options),
      });
      return { data: { id, object: "webhook", deleted: true as const }, meta: { replayed: false } };
    });
  }
}

/**
 * Resend Node SDK drop-in façade backed by Clover V2.
 *
 * ```ts
 * import { Resend } from "@sendclover/sdk/resend";
 * const resend = new Resend(process.env.CLOVER_API_KEY, { baseUrl: "http://127.0.0.1:8080" });
 * const { data, error } = await resend.emails.send({ from, to, subject, html });
 * ```
 */
export class Resend {
  readonly key?: string;
  readonly baseUrl: string;
  readonly userAgent: string;
  readonly emails: ResendEmails;
  readonly batch: ResendBatch;
  readonly domains: ResendDomains;
  readonly apiKeys: ResendApiKeys;
  readonly webhooks: ResendWebhooks;

  constructor(key?: string, options: ResendOptions = {}) {
    const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process
      ?.env;
    const apiKey = key?.trim() || env?.CLOVER_API_KEY || env?.RESEND_API_KEY;
    if (!apiKey)
      throw new TypeError('Missing API key. Pass it to the constructor `new Resend("re_123")`');
    const baseUrl =
      options.baseUrl?.trim() ||
      env?.CLOVER_API_URL ||
      env?.RESEND_BASE_URL ||
      "http://127.0.0.1:8080";
    const clientOptions: ClientOptions = {
      baseUrl,
      apiKey,
      userAgent: options.userAgent ?? "clover-sdk-resend/0.1.0",
      fetch: options.fetch,
      maxRetries: options.maxRetries ?? 0,
      retryBaseDelayMs: options.retryBaseDelayMs,
      maxResponseBodyBytes: options.maxResponseBodyBytes,
      sleep: options.sleep,
    };
    const transport = createTransport(clientOptions);
    this.key = apiKey;
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.userAgent = clientOptions.userAgent!;
    this.emails = new ResendEmails(transport);
    this.batch = new ResendBatch(transport);
    this.domains = new ResendDomains(transport);
    this.apiKeys = new ResendApiKeys(transport);
    this.webhooks = new ResendWebhooks(transport);
  }
}
