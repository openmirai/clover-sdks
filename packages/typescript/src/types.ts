export type JsonObject = { [key: string]: unknown };
export type JsonValue = string | number | boolean | null | JsonObject | JsonValue[];

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
  priority?: string;
  deliver_by?: string | null;
}

export type BatchEmailItem = Omit<SendEmailRequest, "scheduled_at"> & { scheduled_at?: never };

export interface EmailAccepted extends JsonObject {
  id: string;
  status: string;
  scheduled_at?: string | null;
  request_id: string;
}

export interface EmailBatchAccepted extends JsonObject {
  data: Array<{ id: string; status: string }>;
  request_id: string;
}

export interface Pagination {
  page: number;
  limit: number;
  total: number;
  hasNext?: boolean;
  hasPrevious?: boolean;
}

export interface PaginatedData<T> {
  items: T[];
  pagination: Pagination;
}

export type EmailPage = PaginatedData<JsonObject>;

export interface ErrorDetail {
  code: number;
  type: string;
  message: string;
  details?: string;
  fields?: Record<string, string>;
}

export interface ApiErrorBody {
  success: false;
  error: ErrorDetail;
  timestamp?: string;
  requestId?: string;
}

export interface CommonResponse<T> {
  success: boolean;
  message?: string;
  data?: T;
  error?: unknown;
  timestamp?: string;
  requestId?: string;
}

export interface ResponseMeta {
  requestId?: string;
  retryAfter?: number;
  rateLimitRemaining?: number;
  replayed: boolean;
}

export interface ClientOptions {
  /** API origin only (e.g. `http://127.0.0.1:8080`). Paths always use `/api/v1`. */
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
  page?: number;
  limit?: number;
  cursor?: string;
  status?: string;
  domain_id?: string;
  api_key_id?: string;
  request_id?: string;
  [key: string]: string | number | undefined;
}

export interface CreateDomainRequest extends JsonObject {
  name: string;
  provider: string;
  region: string;
  receivingEnabled?: boolean;
  trackingDomain?: string | null;
  tlsPolicy?: string;
}

export interface ConfigureDomainRequest extends JsonObject {
  provider?: string;
  region?: string;
  sendingEnabled?: boolean;
  receivingEnabled?: boolean;
  trackingDomain?: string | null;
  tlsPolicy?: string;
}

export interface CreateAPIKeyRequest extends JsonObject {
  name: string;
  permission?: string;
  domain_id?: string | null;
}

export interface UpdateAPIKeyRequest extends JsonObject {
  name?: string;
  permission?: string;
  domain_id?: string | null;
}

export interface CreateWebhookRequest extends JsonObject {
  url: string;
  description?: string | null;
  subscriptions?: string[];
  enabled?: boolean;
}

export interface UpdateWebhookRequest extends JsonObject {
  url?: string;
  description?: string | null;
  subscriptions?: string[];
  enabled?: boolean;
}

export interface RequestOptions {
  idempotencyKey?: string;
}

export interface PlatformScope {
  accountId: string;
  environmentId: string;
}

export interface PlatformAddress extends JsonObject {
  address: string;
  name?: string;
}

export interface PlatformSendMessageRequest extends JsonObject {
  from: PlatformAddress;
  to: PlatformAddress[];
  cc?: PlatformAddress[];
  bcc?: PlatformAddress[];
  replyTo?: PlatformAddress[];
  subject: string;
  html?: string;
  text?: string;
  headers?: Record<string, string>;
  tags?: Record<string, string>;
  scheduledAt?: string;
}

export interface PlatformMessageAccepted extends JsonObject {
  id: string;
  status: string;
  scheduledAt?: string;
  requestId?: string;
}

export type PlatformMessagePage = PaginatedData<PlatformMessageAccepted>;
