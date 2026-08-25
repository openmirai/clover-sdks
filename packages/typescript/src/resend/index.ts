import { CloverError } from "../errors.js";
import {
  createIdempotencyKey,
  createTransport,
  platformEnvironmentPath,
  type Transport,
  withQuery,
} from "../transport.js";
import type { ClientOptions, JsonObject, PlatformScope, SendEmailRequest } from "../types.js";

export type ResendResult<T> =
  | { data: T; error: null; headers: Record<string, string> | null }
  | {
      data: null;
      error: { message: string; statusCode: number | null; name: string };
      headers: Record<string, string> | null;
    };

export interface ResendOptions {
  baseUrl?: string;
  accountId?: string;
  environmentId?: string;
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
  headers?: Record<string, string>;
  tags?: Tag[];
  attachments?: Array<{
    objectKey: string;
    filename: string;
    sizeBytes: number;
    sha256: string;
    contentType?: string;
    contentId?: string;
  }>;
  scheduledAt?: string;
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
  const request: SendEmailRequest = {
    from: parseAddress(payload.from),
    to: asAddressList(payload.to) ?? [],
    subject: payload.subject,
  };
  const cc = asAddressList(payload.cc);
  const bcc = asAddressList(payload.bcc);
  const reply = asAddressList(payload.replyTo);
  if (cc) request.cc = cc;
  if (bcc) request.bcc = bcc;
  if (reply) request.replyTo = reply;
  if (payload.html !== undefined) request.html = payload.html;
  if (payload.text !== undefined) request.text = payload.text;
  if (payload.headers) request.headers = payload.headers;
  const tags = tagsToMap(payload.tags);
  if (tags) request.tags = tags;
  if (payload.scheduledAt) request.scheduledAt = payload.scheduledAt;
  if (payload.attachments?.length) {
    request.attachments = payload.attachments.map((attachment) => ({
      objectKey: attachment.objectKey,
      filename: attachment.filename,
      contentType: attachment.contentType ?? "application/octet-stream",
      disposition: attachment.contentId ? "inline" : "attachment",
      ...(attachment.contentId ? { contentId: attachment.contentId } : {}),
      sizeBytes: attachment.sizeBytes,
      sha256: attachment.sha256,
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
  constructor(
    private readonly transport: Transport,
    private readonly scope: PlatformScope,
  ) {}

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
      const { data, meta } = await this.transport.request<{ id: string }>(
        platformEnvironmentPath(this.scope, "/messages"),
        {
          method: "POST",
          body: toCloverSendRequest(payload),
          idempotencyKey: resolveIdempotencyKey(options),
        },
      );
      return { data: { id: String(data.id) }, meta };
    });
  }

  get(id: string): Promise<ResendResult<JsonObject>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<JsonObject>(
        platformEnvironmentPath(this.scope, `/messages/${encodeURIComponent(id)}`),
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
      }>(withQuery(platformEnvironmentPath(this.scope, "/messages"), options), { method: "GET" });
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
        platformEnvironmentPath(this.scope, `/messages/${encodeURIComponent(payload.id)}/schedule`),
        {
          method: "POST",
          body: { scheduledAt: payload.scheduledAt },
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
        platformEnvironmentPath(this.scope, `/messages/${encodeURIComponent(id)}/cancel`),
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
  constructor(
    private readonly transport: Transport,
    private readonly scope: PlatformScope,
  ) {}

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
        const { scheduledAt: _scheduledAt, ...rest } = body;
        return rest;
      });
      const { data, meta } = await this.transport.request<{ items?: Array<{ id: string }> }>(
        platformEnvironmentPath(this.scope, "/messages/batch"),
        {
          method: "POST",
          body: { items },
          idempotencyKey: resolveIdempotencyKey(options),
        },
      );
      return {
        data: { data: (data.items ?? []).map((item) => ({ id: String(item.id) })) },
        meta,
      };
    });
  }
}

class ResendDomains {
  constructor(
    private readonly transport: Transport,
    private readonly scope: PlatformScope,
  ) {}

  create(
    payload: {
      name: string;
      providerBindingId: string;
    },
    options?: IdempotentOptions,
  ): Promise<ResendResult<JsonObject>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<JsonObject>(
        platformEnvironmentPath(this.scope, "/domains"),
        {
          method: "POST",
          body: {
            domain: payload.name,
            providerBindingId: payload.providerBindingId,
          },
          idempotencyKey: resolveIdempotencyKey(options),
        },
      );
      return { data, meta };
    });
  }

  list(
    options: { page?: number; limit?: number } = {},
  ): Promise<ResendResult<{ object: "list"; data: JsonObject[] }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{ items?: JsonObject[] }>(
        withQuery(platformEnvironmentPath(this.scope, "/domains"), options),
        { method: "GET" },
      );
      return { data: { object: "list", data: data.items ?? [] }, meta };
    });
  }

  verify(id: string, options?: IdempotentOptions): Promise<ResendResult<JsonObject>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<JsonObject>(
        platformEnvironmentPath(this.scope, `/domains/${encodeURIComponent(id)}/verify`),
        {
          method: "POST",
          idempotencyKey: resolveIdempotencyKey(options),
        },
      );
      return { data, meta };
    });
  }
}

class ResendWebhooks {
  constructor(
    private readonly transport: Transport,
    private readonly scope: PlatformScope,
  ) {}

  create(
    payload: { endpoint: string; events: string[] },
    options?: IdempotentOptions,
  ): Promise<ResendResult<{ id: string; signing_secret: string; object: "webhook" }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{
        endpoint?: { id?: string };
        secret?: string;
      }>(platformEnvironmentPath(this.scope, "/webhooks"), {
        method: "POST",
        body: {
          url: payload.endpoint,
          subscriptions: payload.events,
        },
        idempotencyKey: resolveIdempotencyKey(options),
      });
      return {
        data: {
          object: "webhook",
          id: String(data.endpoint?.id ?? ""),
          signing_secret: String(data.secret ?? ""),
        },
        meta,
      };
    });
  }

  get(id: string): Promise<ResendResult<JsonObject>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<JsonObject>(
        platformEnvironmentPath(this.scope, `/webhooks/${encodeURIComponent(id)}`),
        { method: "GET" },
      );
      return { data, meta };
    });
  }

  list(
    options: { cursor?: string; limit?: number; enabled?: boolean } = {},
  ): Promise<ResendResult<{ object: "list"; data: JsonObject[] }>> {
    return asResult(async () => {
      const { data, meta } = await this.transport.request<{ items?: JsonObject[] }>(
        withQuery(platformEnvironmentPath(this.scope, "/webhooks"), options),
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
      await this.transport.request(
        platformEnvironmentPath(this.scope, `/webhooks/${encodeURIComponent(id)}`),
        {
          method: "DELETE",
          idempotencyKey: resolveIdempotencyKey(options),
        },
      );
      return { data: { id, object: "webhook", deleted: true as const }, meta: { replayed: false } };
    });
  }
}

/**
 * Resend Node SDK drop-in façade backed by Clover V2.
 *
 * ```ts
 * import { Resend } from "@sendclover/sdk/resend";
 * const resend = new Resend(process.env.CLOVER_API_KEY, {
 *   baseUrl: "http://127.0.0.1:8080",
 *   accountId: process.env.CLOVER_ACCOUNT_ID!,
 *   environmentId: process.env.CLOVER_ENVIRONMENT_ID!,
 * });
 * const { data, error } = await resend.emails.send({ from, to, subject, html });
 * ```
 */
export class Resend {
  readonly key?: string;
  readonly baseUrl: string;
  readonly accountId: string;
  readonly environmentId: string;
  readonly userAgent: string;
  readonly emails: ResendEmails;
  readonly batch: ResendBatch;
  readonly domains: ResendDomains;
  readonly webhooks: ResendWebhooks;

  constructor(key?: string, options: ResendOptions = {}) {
    const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process
      ?.env;
    const apiKey = key?.trim() || env?.CLOVER_API_KEY || env?.RESEND_API_KEY;
    if (!apiKey)
      throw new TypeError('Missing API key. Pass it to the constructor `new Resend("re_123")`');
    const accountId = options.accountId?.trim() || env?.CLOVER_ACCOUNT_ID?.trim();
    const environmentId = options.environmentId?.trim() || env?.CLOVER_ENVIRONMENT_ID?.trim();
    if (!accountId || !environmentId) {
      throw new TypeError("accountId and environmentId are required");
    }
    const scope = { accountId, environmentId };
    const baseUrl =
      options.baseUrl?.trim() ||
      env?.CLOVER_API_URL ||
      env?.RESEND_BASE_URL ||
      "http://127.0.0.1:8080";
    const clientOptions: ClientOptions = {
      baseUrl,
      apiKey,
      accountId,
      environmentId,
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
    this.accountId = accountId;
    this.environmentId = environmentId;
    this.userAgent = clientOptions.userAgent!;
    this.emails = new ResendEmails(transport, scope);
    this.batch = new ResendBatch(transport, scope);
    this.domains = new ResendDomains(transport, scope);
    this.webhooks = new ResendWebhooks(transport, scope);
  }
}
