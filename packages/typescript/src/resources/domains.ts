import { platformEnvironmentPath, type Transport, withQuery } from "../transport.js";
import type {
  CreateDomainRequest,
  DomainListPayload,
  DomainOnboardingResponse,
  DomainResponse,
  PlatformScope,
  RequestOptions,
} from "../types.js";

export interface ListDomainsOptions {
  page?: number;
  limit?: number;
  [key: string]: string | number | undefined;
}

const domainsPath = (scope: PlatformScope, suffix = ""): string =>
  platformEnvironmentPath(scope, `/domains${suffix}`);

export class DomainsResource {
  constructor(
    private readonly transport: Transport,
    private readonly scope: PlatformScope,
  ) {}

  async create(
    request: CreateDomainRequest,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<DomainOnboardingResponse> {
    const { data } = await this.transport.request<DomainOnboardingResponse>(
      domainsPath(this.scope),
      {
        method: "POST",
        body: request,
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }

  async list(options: ListDomainsOptions = {}): Promise<DomainListPayload> {
    const { data } = await this.transport.request<DomainListPayload>(
      withQuery(domainsPath(this.scope), options),
      { method: "GET" },
    );
    return data;
  }

  async verify(
    id: string,
    options: RequestOptions & { idempotencyKey: string },
  ): Promise<DomainResponse> {
    const { data } = await this.transport.request<DomainResponse>(
      domainsPath(this.scope, `/${encodeURIComponent(id)}/verify`),
      {
        method: "POST",
        idempotencyKey: options.idempotencyKey,
      },
    );
    return data;
  }
}
