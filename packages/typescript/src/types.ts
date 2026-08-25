export type JsonObject = { [key: string]: unknown };
export type JsonValue = string | number | boolean | null | JsonObject | JsonValue[];

export interface EmailAddress {
  address: string;
  name?: string | null;
}

export interface AttachmentRequest extends JsonObject {
  objectKey: string;
  filename: string;
  contentType: string;
  disposition?: string;
  contentId?: string;
  sizeBytes: number;
  sha256: string;
}

export interface PlatformSendMessageRequest extends JsonObject {
  from: EmailAddress;
  to: EmailAddress[];
  cc?: EmailAddress[];
  bcc?: EmailAddress[];
  replyTo?: EmailAddress[];
  subject: string;
  html?: string | null;
  text?: string | null;
  metadata?: Record<string, unknown>;
  attachments?: AttachmentRequest[];
  headers?: Record<string, string>;
  tags?: Record<string, string>;
  scheduledAt?: string | null;
}

export type SendEmailRequest = PlatformSendMessageRequest;
export type BatchEmailItem = Omit<SendEmailRequest, "scheduledAt"> & { scheduledAt?: never };

export interface EmailAccepted extends JsonObject {
  id: string;
  status: string;
  scheduledAt?: string | null;
  requestId?: string;
}

export interface EmailBatchAccepted extends JsonObject {
  items: Array<{ id: string; status: string }>;
  requestId?: string;
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
  /** API origin only (e.g. `http://127.0.0.1:8080`). */
  baseUrl: string;
  apiKey: string;
  /** Client account used in scoped platform paths. */
  accountId: string;
  /** Environment used in scoped platform paths. */
  environmentId: string;
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
  [key: string]: string | number | undefined;
}

export interface CreateDomainRequest extends JsonObject {
  domain: string;
  providerBindingId: string;
}

export interface DomainResponse extends JsonObject {
  id: string;
  name: string;
  providerBindingId: string;
  verificationState: string;
  createdAt?: string;
  updatedAt?: string;
  lastVerifiedAt?: string;
  requiredRecords?: JsonObject;
}

export interface DomainListPayload {
  items: DomainResponse[];
  pagination: Pagination;
}

export interface DomainOnboardingResponse extends JsonObject {
  domain: DomainResponse;
  availableSources: JsonObject[];
  domainConnectTemplate?: JsonObject;
}

export interface CreateWebhookRequest extends JsonObject {
  url: string;
  description?: string | null;
  subscriptions: string[];
  enabled?: boolean;
}

export interface UpdateWebhookRequest extends JsonObject {
  url?: string;
  description?: string | null;
  subscriptions?: string[];
  enabled?: boolean;
}

export interface WebhookListPayload {
  items: JsonObject[];
  next_cursor?: string;
}

export interface RequestOptions {
  idempotencyKey?: string;
}

export interface PlatformScope {
  accountId: string;
  environmentId: string;
}

export type PlatformAddress = EmailAddress;

export interface PlatformMessageAccepted extends JsonObject {
  id: string;
  status: string;
  scheduledAt?: string;
  requestId?: string;
}

export interface PlatformMessageSummary extends JsonObject {
  id: string;
  environmentId: string;
  from: EmailAddress;
  toCount: number;
  subject: string;
  status: string;
  scheduledAt?: string;
  providerMessageId?: string;
  requestId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface PlatformMessageDetail extends PlatformMessageSummary {
  html?: string;
  text?: string;
  replyTo: EmailAddress[];
  headers: Record<string, string>;
  tags: Record<string, string>;
  metadata: Record<string, unknown>;
  recipients: JsonObject[];
  attachments: JsonObject[];
  attempts: JsonObject[];
  events: JsonObject[];
}

export type EmailPage = PaginatedData<PlatformMessageSummary>;
export type PlatformMessagePage = EmailPage;
