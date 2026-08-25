export { CloverClient, CloverError } from "./client.js";
export type {
  AttachmentRequest,
  BatchEmailItem,
  ClientOptions,
  CommonResponse,
  CreateDomainRequest,
  CreateWebhookRequest,
  DomainListPayload,
  DomainOnboardingResponse,
  DomainResponse,
  EmailAccepted,
  EmailAddress,
  EmailBatchAccepted,
  EmailPage,
  ErrorDetail,
  JsonObject,
  ListEmailsOptions,
  PaginatedData,
  Pagination,
  PlatformAddress,
  PlatformMessageAccepted,
  PlatformMessageDetail,
  PlatformMessagePage,
  PlatformMessageSummary,
  PlatformScope,
  PlatformSendMessageRequest,
  RequestOptions,
  ResponseMeta,
  SendEmailRequest,
  UpdateWebhookRequest,
  WebhookListPayload,
} from "./types.js";
export { DomainsResource } from "./resources/domains.js";
export { EmailsResource } from "./resources/emails.js";
export { WebhooksResource } from "./resources/webhooks.js";
export { PlatformMessagesResource } from "./resources/platform-messages.js";
