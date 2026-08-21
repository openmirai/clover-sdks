import type { ApiErrorBody, ErrorDetail, ResponseMeta } from "./types.js";

export interface LegacyProblem {
  type?: string;
  title: string;
  status: number;
  code: string;
  detail?: string;
  request_id?: string;
  field_errors?: Record<string, string[]>;
  [key: string]: unknown;
}

export class CloverError extends Error {
  readonly status: number;
  readonly error?: ErrorDetail;
  readonly meta: ResponseMeta;
  /** @deprecated Prefer `error` — retained for callers that still expect Problem-shaped fields. */
  readonly problem?: LegacyProblem;

  constructor(
    message: string,
    status: number,
    detail: ErrorDetail | undefined,
    meta: ResponseMeta,
  ) {
    super(message);
    this.name = "CloverError";
    this.status = status;
    this.error = detail;
    this.meta = meta;
    if (detail) {
      this.problem = {
        type: detail.type,
        title: detail.message,
        status,
        code: String(detail.code),
        detail: detail.details,
        request_id: meta.requestId,
        field_errors: detail.fields
          ? Object.fromEntries(Object.entries(detail.fields).map(([k, v]) => [k, [v]]))
          : undefined,
      };
    }
  }
}

export function isApiErrorBody(value: unknown): value is ApiErrorBody {
  if (!value || typeof value !== "object") return false;
  const body = value as ApiErrorBody;
  return (
    body.success === false &&
    typeof body.error === "object" &&
    body.error !== null &&
    typeof body.error.message === "string"
  );
}
