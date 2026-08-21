import { CloverError, isApiErrorBody } from "./errors.js";
import type { ClientOptions, CommonResponse, JsonObject, ResponseMeta } from "./types.js";

const RETRYABLE_STATUS = new Set([408, 425, 429, 500, 502, 503, 504]);
export const IDEMPOTENCY_KEY = /^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}(?![\s\S])/;
export const DEFAULT_USER_AGENT = "clover-sdk-typescript/0.1.0";
export const DEFAULT_MAX_RESPONSE_BODY_BYTES = 4 * 1024 * 1024;
export const API_PREFIX = "/api/v1";

class ResponseBodyLimitError extends Error {}

export function normalizeBaseUrl(baseUrl: string): string {
  const trimmed = baseUrl?.trim();
  if (!trimmed) throw new TypeError("baseUrl is required");
  let parsed: URL;
  try {
    parsed = new URL(trimmed);
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
  ) {
    throw new TypeError("baseUrl must be an absolute http(s) URL without userinfo/query/fragment");
  }
  return trimmed.replace(/\/$/, "");
}

export function createIdempotencyKey(): string {
  const bytes = new Uint8Array(16);
  globalThis.crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

export function createRequestId(): string {
  const bytes = new Uint8Array(8);
  globalThis.crypto.getRandomValues(bytes);
  return `req_${Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("")}`;
}

export function assertIdempotencyKey(
  key: string | undefined,
  required: boolean,
): string | undefined {
  if (!key) {
    if (required)
      throw new TypeError("idempotency key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$");
    return undefined;
  }
  if (!IDEMPOTENCY_KEY.test(key)) {
    throw new TypeError("idempotency key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$");
  }
  return key;
}

export interface Transport {
  request<T>(
    path: string,
    options: {
      method: string;
      body?: JsonObject | JsonObject[];
      idempotencyKey?: string;
      requireIdempotency?: boolean;
    },
  ): Promise<{ data: T; meta: ResponseMeta; raw: JsonObject }>;
}

export function createTransport(options: ClientOptions): Transport {
  const baseUrl = normalizeBaseUrl(options.baseUrl);
  if (!options.apiKey?.trim()) throw new TypeError("apiKey is required");
  const apiKey = options.apiKey;
  const userAgent = options.userAgent ?? DEFAULT_USER_AGENT;
  const fetchImpl = options.fetch ?? globalThis.fetch.bind(globalThis);
  const maxRetries = Math.max(0, Math.min(3, Math.trunc(options.maxRetries ?? 2)));
  const retryBaseDelayMs = Math.max(0, options.retryBaseDelayMs ?? 100);
  const maxResponseBodyBytes = Math.max(
    1,
    Math.trunc(options.maxResponseBodyBytes ?? DEFAULT_MAX_RESPONSE_BODY_BYTES),
  );
  const sleep =
    options.sleep ?? ((ms: number) => new Promise((resolve) => setTimeout(resolve, ms)));

  return {
    async request<T>(
      path: string,
      requestOptions: {
        method: string;
        body?: JsonObject | JsonObject[];
        idempotencyKey?: string;
        requireIdempotency?: boolean;
      },
    ) {
      const requireIdempotency =
        requestOptions.requireIdempotency ?? requestOptions.method !== "GET";
      const idempotencyKey = assertIdempotencyKey(
        requestOptions.idempotencyKey,
        requireIdempotency,
      );
      const safeToRetry = requestOptions.method === "GET" || Boolean(idempotencyKey);
      const headers = new Headers({
        Accept: "application/json",
        Authorization: `Bearer ${apiKey}`,
        "User-Agent": userAgent,
        "X-Request-ID": createRequestId(),
      });
      if (requestOptions.body !== undefined) headers.set("Content-Type", "application/json");
      if (idempotencyKey !== undefined) headers.set("Idempotency-Key", idempotencyKey);

      let attempt = 0;
      while (true) {
        const response = await fetchImpl(`${baseUrl}${path}`, {
          method: requestOptions.method,
          headers,
          body: requestOptions.body === undefined ? undefined : JSON.stringify(requestOptions.body),
        });
        const meta = metadata(response.headers);
        let rawText: string;
        try {
          rawText = await readBoundedBody(response, maxResponseBodyBytes);
        } catch (error) {
          if (error instanceof ResponseBodyLimitError) {
            throw new CloverError(
              "Clover response body exceeds the configured limit",
              response.status,
              undefined,
              meta,
            );
          }
          throw error;
        }

        let parsed: JsonObject = {};
        if (rawText) {
          try {
            parsed = JSON.parse(rawText) as JsonObject;
          } catch {
            parsed = { raw: rawText };
          }
        }

        if (response.ok) {
          const envelope = parsed as unknown as CommonResponse<T>;
          if (typeof envelope.success === "boolean") {
            if (!envelope.success) {
              throw cloverErrorFromBody(response.status, parsed, meta);
            }
            const data = (envelope.data ?? {}) as T;
            return {
              data,
              meta: {
                ...meta,
                requestId: meta.requestId ?? envelope.requestId,
              },
              raw: parsed,
            };
          }
          // Flat legacy body — return as-is so older mocks still work.
          return { data: parsed as T, meta, raw: parsed };
        }

        if (safeToRetry && attempt < maxRetries && RETRYABLE_STATUS.has(response.status)) {
          const retryAfter = meta.retryAfter;
          const delay =
            retryAfter === undefined ? retryBaseDelayMs * 2 ** attempt : retryAfter * 1000;
          attempt += 1;
          await sleep(delay);
          continue;
        }

        throw cloverErrorFromBody(response.status, parsed, meta);
      }
    },
  };
}

function cloverErrorFromBody(status: number, parsed: JsonObject, meta: ResponseMeta): CloverError {
  if (isApiErrorBody(parsed)) {
    return new CloverError(parsed.error.message, status, parsed.error, {
      ...meta,
      requestId: meta.requestId ?? parsed.requestId,
    });
  }
  const title =
    (typeof parsed.message === "string" && parsed.message) ||
    (typeof parsed.title === "string" && parsed.title) ||
    `Clover request failed (${status})`;
  return new CloverError(title, status, undefined, meta);
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

export function withQuery(
  path: string,
  options: Record<string, string | number | undefined>,
): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(options)) {
    if (value !== undefined) query.set(key, String(value));
  }
  const suffix = query.toString() ? `?${query}` : "";
  return `${path}${suffix}`;
}
