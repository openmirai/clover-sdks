export type JsonObject = { [key: string]: unknown };

export interface Problem extends JsonObject {
  type: string;
  title: string;
  status: number;
  code: string;
  detail?: string | null;
  request_id: string;
  field_errors?: Record<string, string[]>;
}

export interface EmailAddress {
  address: string;
  name?: string | null;
}
export interface SendEmailRequest extends JsonObject {
  from: EmailAddress;
  to: EmailAddress[];
  cc?: EmailAddress[];
  bcc?: EmailAddress[];
  reply_to?: EmailAddress[];
  subject: string;
  html?: string | null;
  text?: string | null;
  attachments?: JsonObject[];
  headers?: Record<string, string>;
  tags?: Record<string, string>;
  scheduled_at?: string | null;
}
export type BatchEmailItem = Omit<SendEmailRequest, "scheduled_at"> & { scheduled_at?: never };
export interface EmailAccepted extends JsonObject {
  id: string;
  status: string;
  scheduled_at?: string | null;
  request_id: string;
}
export interface EmailPage extends JsonObject {
  data: JsonObject[];
  next_cursor?: string | null;
}

export interface ResponseMeta {
  requestId?: string;
  retryAfter?: number;
  rateLimitRemaining?: number;
  replayed: boolean;
}

export class CloverError extends Error {
  readonly status: number;
  readonly problem?: Problem;
  readonly meta: ResponseMeta;
  constructor(message: string, status: number, problem: Problem | undefined, meta: ResponseMeta) {
    super(message);
    this.name = "CloverError";
    this.status = status;
    this.problem = problem;
    this.meta = meta;
  }
}

export interface ClientOptions {
  baseUrl: string;
  apiKey: string;
  userAgent?: string;
  fetch?: typeof globalThis.fetch;
  maxRetries?: number;
  retryBaseDelayMs?: number;
  maxResponseBodyBytes?: number;
  sleep?: (milliseconds: number) => Promise<void>;
}

export interface ListEmailsOptions {
  cursor?: string;
  limit?: number;
  status?: string;
  [key: string]: string | number | undefined;
}

const RETRYABLE_STATUS = new Set([408, 425, 429, 500, 502, 503, 504]);
// The final negative lookahead makes the end anchor exact (JavaScript's `$`
// otherwise accepts a trailing newline).
const IDEMPOTENCY_KEY = /^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}(?![\s\S])/;
const DEFAULT_USER_AGENT = "clover-sdk-typescript/0.1.0";
const DEFAULT_MAX_RESPONSE_BODY_BYTES = 4 * 1024 * 1024;

class ResponseBodyLimitError extends Error {}

export class CloverClient {
  private readonly baseUrl: string;
  private readonly apiKey: string;
  private readonly userAgent: string;
  private readonly fetchImpl: typeof globalThis.fetch;
  private readonly maxRetries: number;
  private readonly retryBaseDelayMs: number;
  private readonly maxResponseBodyBytes: number;
  private readonly sleep: (milliseconds: number) => Promise<void>;

  constructor(options: ClientOptions) {
    const baseUrl = options.baseUrl?.trim();
    if (!baseUrl) throw new TypeError("baseUrl is required");
    let parsed: URL;
    try {
      parsed = new URL(baseUrl);
    } catch {
      throw new TypeError("baseUrl must be an absolute http(s) URL");
    }
    if (
      !parsed.hostname ||
      !["http:", "https:"].includes(parsed.protocol) ||
      parsed.username ||
      parsed.password ||
      parsed.search ||
      parsed.hash
    )
      throw new TypeError(
        "baseUrl must be an absolute http(s) URL without userinfo/query/fragment",
      );
    if (!options.apiKey?.trim()) throw new TypeError("apiKey is required");
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.apiKey = options.apiKey;
    this.userAgent = options.userAgent ?? DEFAULT_USER_AGENT;
    this.fetchImpl = options.fetch ?? globalThis.fetch.bind(globalThis);
    this.maxRetries = Math.max(0, Math.min(3, Math.trunc(options.maxRetries ?? 2)));
    this.retryBaseDelayMs = Math.max(0, options.retryBaseDelayMs ?? 100);
    this.maxResponseBodyBytes = Math.max(
      1,
      Math.trunc(options.maxResponseBodyBytes ?? DEFAULT_MAX_RESPONSE_BODY_BYTES),
    );
    this.sleep = options.sleep ?? ((ms) => new Promise((resolve) => setTimeout(resolve, ms)));
  }

  async send(request: SendEmailRequest, idempotencyKey: string): Promise<EmailAccepted> {
    return this.request<EmailAccepted>("/v1/emails", {
      method: "POST",
      body: request,
      idempotencyKey,
    });
  }

  async sendBatch(items: BatchEmailItem[], idempotencyKey: string): Promise<JsonObject> {
    const sanitized = items.map(({ scheduled_at: _scheduledAt, ...item }) => item);
    return this.request<JsonObject>("/v1/emails/batch", {
      method: "POST",
      body: { items: sanitized },
      idempotencyKey,
    });
  }

  async schedule(id: string, scheduledAt: string, idempotencyKey: string): Promise<EmailAccepted> {
    return this.request<EmailAccepted>(`/v1/emails/${encodeURIComponent(id)}/schedule`, {
      method: "POST",
      body: { scheduled_at: scheduledAt },
      idempotencyKey,
    });
  }

  async cancel(id: string, idempotencyKey: string): Promise<JsonObject> {
    return this.request<JsonObject>(`/v1/emails/${encodeURIComponent(id)}/cancel`, {
      method: "POST",
      idempotencyKey,
    });
  }

  async get(id: string): Promise<JsonObject> {
    return this.request<JsonObject>(`/v1/emails/${encodeURIComponent(id)}`, { method: "GET" });
  }

  async list(options: ListEmailsOptions = {}): Promise<EmailPage> {
    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(options))
      if (value !== undefined) query.set(key, String(value));
    const suffix = query.toString() ? `?${query}` : "";
    return this.request<EmailPage>(`/v1/emails${suffix}`, { method: "GET" });
  }

  private async request<T extends JsonObject>(
    path: string,
    options: {
      method: string;
      body?: JsonObject;
      idempotencyKey?: string;
    },
  ): Promise<T> {
    if (
      options.method !== "GET" &&
      (!options.idempotencyKey || !IDEMPOTENCY_KEY.test(options.idempotencyKey))
    ) {
      throw new TypeError("idempotency key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$");
    }
    const safeToRetry = options.method === "GET" || Boolean(options.idempotencyKey);
    const headers = new Headers({
      Accept: "application/json, application/problem+json",
      Authorization: `Bearer ${this.apiKey}`,
      "User-Agent": this.userAgent,
    });
    if (options.body !== undefined) {
      headers.set("Content-Type", "application/json");
    }
    if (options.idempotencyKey !== undefined)
      headers.set("Idempotency-Key", options.idempotencyKey);
    let attempt = 0;
    while (true) {
      const response = await this.fetchImpl(`${this.baseUrl}${path}`, {
        method: options.method,
        headers,
        body: options.body === undefined ? undefined : JSON.stringify(options.body),
      });
      const meta = metadata(response.headers);
      let raw: string;
      try {
        raw = await readBoundedBody(response, this.maxResponseBodyBytes);
      } catch (error) {
        if (error instanceof ResponseBodyLimitError)
          throw new CloverError(
            "Clover response body exceeds the configured limit",
            response.status,
            undefined,
            meta,
          );
        throw error;
      }
      let parsed: JsonObject = {};
      if (raw) {
        try {
          parsed = JSON.parse(raw) as JsonObject;
        } catch {
          parsed = { raw };
        }
      }
      if (response.ok) return parsed as T;
      if (safeToRetry && attempt < this.maxRetries && RETRYABLE_STATUS.has(response.status)) {
        const retryAfter = meta.retryAfter;
        const delay =
          retryAfter === undefined ? this.retryBaseDelayMs * 2 ** attempt : retryAfter * 1000;
        attempt += 1;
        await this.sleep(delay);
        continue;
      }
      throw new CloverError(
        (parsed.title as string | undefined) ?? `Clover request failed (${response.status})`,
        response.status,
        isProblem(parsed) ? (parsed as Problem) : undefined,
        meta,
      );
    }
  }
}

async function readBoundedBody(response: Response, limit: number): Promise<string> {
  if (!response.body) {
    const raw = await response.text();
    if (new TextEncoder().encode(raw).byteLength > limit) throw new ResponseBodyLimitError();
    return raw;
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      if (value) {
        total += value.byteLength;
        if (total > limit) {
          await reader.cancel();
          throw new ResponseBodyLimitError();
        }
        chunks.push(value);
      }
    }
  } finally {
    reader.releaseLock();
  }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder().decode(bytes);
}

function metadata(headers: Headers): ResponseMeta {
  const retryHeader = headers.get("retry-after");
  const retry = retryHeader === null ? Number.NaN : Number(retryHeader);
  const remaining = Number(headers.get("x-ratelimit-remaining"));
  return {
    requestId: headers.get("x-request-id") ?? undefined,
    retryAfter: Number.isFinite(retry) && retry >= 0 ? retry : undefined,
    rateLimitRemaining: Number.isFinite(remaining) ? remaining : undefined,
    replayed: headers.get("idempotency-replayed")?.toLowerCase() === "true",
  };
}

function isProblem(value: JsonObject): value is Problem {
  return (
    typeof value.type === "string" &&
    typeof value.title === "string" &&
    typeof value.status === "number" &&
    typeof value.code === "string"
  );
}
