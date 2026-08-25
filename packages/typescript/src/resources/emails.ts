import type { Transport } from "../transport.js";
import type {
  BatchEmailItem,
  EmailAccepted,
  EmailBatchAccepted,
  EmailPage,
  ListEmailsOptions,
  PlatformScope,
  PlatformMessageDetail,
  PlatformMessageSummary,
  RequestOptions,
  SendEmailRequest,
} from "../types.js";
import { PlatformMessagesResource } from "./platform-messages.js";

export class EmailsResource {
  private readonly messages: PlatformMessagesResource;

  constructor(
    transport: Transport,
    private readonly scope: PlatformScope,
  ) {
    this.messages = new PlatformMessagesResource(transport);
  }

  send(
    request: SendEmailRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<EmailAccepted> {
    return this.messages.send(this.scope, request, options);
  }

  sendBatch(
    items: BatchEmailItem[],
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<EmailBatchAccepted> {
    return this.messages.sendBatch(this.scope, items, options);
  }

  schedule(
    id: string,
    scheduledAt: string,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<EmailAccepted> {
    return this.messages.schedule(this.scope, id, scheduledAt, options);
  }

  cancel(
    id: string,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<PlatformMessageSummary> {
    return this.messages.cancel(this.scope, id, options);
  }

  get(id: string): Promise<PlatformMessageDetail> {
    return this.messages.get(this.scope, id);
  }

  list(options: ListEmailsOptions = {}): Promise<EmailPage> {
    return this.messages.list(this.scope, options);
  }
}
