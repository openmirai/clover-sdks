"""A small, standard-library-only Clover API client.

The transport seam keeps tests deterministic and lets applications supply a
proxy, tracing transport, or an async adapter without changing the contract.
"""

from __future__ import annotations

import json
import re
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass, field
from typing import Any, Callable, Mapping, MutableMapping, Sequence

Problem = dict[str, Any]
IDEMPOTENCY_KEY = re.compile(r"\A[A-Za-z0-9][A-Za-z0-9._:-]{7,127}\Z")
DEFAULT_MAX_RESPONSE_BODY_BYTES = 4 * 1024 * 1024


@dataclass(frozen=True)
class HttpResponse:
    status: int
    headers: Mapping[str, str] = field(default_factory=dict)
    body: bytes = b""


class CloverError(RuntimeError):
    """A non-2xx response, with a decoded problem document when available."""

    def __init__(self, status: int, message: str, problem: Problem | None = None, metadata: Mapping[str, Any] | None = None):
        super().__init__(message)
        self.status = status
        self.problem = problem
        self.metadata = dict(metadata or {})


class ResponseBodyLimitExceeded(RuntimeError):
    """Raised by the standard transport before an oversized body is parsed."""

    def __init__(self, status: int, headers: Mapping[str, str]):
        super().__init__("Clover response body exceeds the configured limit")
        self.status = status
        self.headers = headers


Transport = Callable[[str, str, Mapping[str, str], bytes | None, float], HttpResponse]


def _read_bounded(stream: Any, limit: int, status: int, headers: Mapping[str, str]) -> bytes:
    chunks: list[bytes] = []
    total = 0
    while total <= limit:
        chunk = stream.read(min(64 * 1024, limit + 1 - total))
        if not chunk:
            break
        chunks.append(chunk)
        total += len(chunk)
    if total > limit:
        raise ResponseBodyLimitExceeded(status, headers)
    return b"".join(chunks)


def _urllib_transport(
    method: str,
    url: str,
    headers: Mapping[str, str],
    body: bytes | None,
    timeout: float,
    *,
    max_response_body_bytes: int = DEFAULT_MAX_RESPONSE_BODY_BYTES,
) -> HttpResponse:
    request = urllib.request.Request(url, data=body, headers=dict(headers), method=method)
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            response_headers = dict(response.headers.items())
            return HttpResponse(
                response.status,
                response_headers,
                _read_bounded(response, max_response_body_bytes, response.status, response_headers),
            )
    except urllib.error.HTTPError as error:
        response_headers = dict(error.headers.items())
        try:
            body_bytes = _read_bounded(error, max_response_body_bytes, error.code, response_headers)
        finally:
            error.close()
        return HttpResponse(error.code, response_headers, body_bytes)


class CloverClient:
    """Clover client for send/batch/schedule/cancel/get/list operations."""

    RETRYABLE = frozenset({408, 425, 429, 500, 502, 503, 504})

    def __init__(self, base_url: str, api_key: str, *, user_agent: str = "clover-sdk-python/0.1.0",
                 max_retries: int = 2, retry_base_delay: float = 0.1, timeout: float = 30.0,
                 max_response_body_bytes: int = DEFAULT_MAX_RESPONSE_BODY_BYTES,
                 transport: Transport | None = None, sleep: Callable[[float], None] = time.sleep):
        candidate = base_url.strip()
        parsed = urllib.parse.urlsplit(candidate)
        if parsed.scheme not in {"http", "https"} or not parsed.netloc or parsed.username or parsed.password or parsed.query or parsed.fragment:
            raise ValueError("base_url must be an absolute http(s) URL without userinfo/query/fragment")
        if not api_key.strip():
            raise ValueError("api_key is required")
        self.base_url = candidate.rstrip("/")
        self.api_key = api_key
        self.user_agent = user_agent
        self.max_retries = max(0, min(3, int(max_retries)))
        self.retry_base_delay = max(0.0, retry_base_delay)
        self.timeout = timeout
        self.max_response_body_bytes = max(1, int(max_response_body_bytes))
        self.transport = transport or (
            lambda method, url, headers, body, request_timeout: _urllib_transport(
                method,
                url,
                headers,
                body,
                request_timeout,
                max_response_body_bytes=self.max_response_body_bytes,
            )
        )
        self.sleep = sleep
        self.last_metadata: dict[str, Any] = {}

    def send(self, request: Mapping[str, Any], idempotency_key: str) -> dict[str, Any]:
        return self._request("POST", "/v1/emails", request, idempotency_key)

    def send_batch(self, items: Sequence[Mapping[str, Any]], idempotency_key: str) -> dict[str, Any]:
        sanitized = [{key: value for key, value in item.items() if key != "scheduled_at"} for item in items]
        return self._request("POST", "/v1/emails/batch", {"items": sanitized}, idempotency_key)

    def schedule(self, email_id: str, scheduled_at: str, idempotency_key: str) -> dict[str, Any]:
        return self._request("POST", f"/v1/emails/{self._path_segment(email_id)}/schedule", {"scheduled_at": scheduled_at}, idempotency_key)

    def cancel(self, email_id: str, idempotency_key: str) -> dict[str, Any]:
        return self._request("POST", f"/v1/emails/{self._path_segment(email_id)}/cancel", None, idempotency_key)

    def get(self, email_id: str) -> dict[str, Any]:
        return self._request("GET", f"/v1/emails/{self._path_segment(email_id)}")

    def list(self, *, cursor: str | None = None, limit: int | None = None, status: str | None = None,
             **filters: str | int) -> dict[str, Any]:
        params: dict[str, str] = {key: str(value) for key, value in filters.items()}
        if cursor is not None:
            params["cursor"] = cursor
        if limit is not None:
            params["limit"] = str(limit)
        if status is not None:
            params["status"] = status
        query = "&".join(f"{urllib.parse.quote(k, safe='')}={urllib.parse.quote(v, safe='')}" for k, v in params.items())
        return self._request("GET", "/v1/emails" + (f"?{query}" if query else ""))

    def _request(self, method: str, path: str, payload: Mapping[str, Any] | None = None,
                 idempotency_key: str | None = None) -> dict[str, Any]:
        if method != "GET" and (not idempotency_key or not IDEMPOTENCY_KEY.fullmatch(idempotency_key)):
            raise ValueError("idempotency_key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$")
        headers: MutableMapping[str, str] = {
            "Accept": "application/json, application/problem+json",
            "Authorization": f"Bearer {self.api_key}",
            "User-Agent": self.user_agent,
        }
        body: bytes | None = None
        if payload is not None:
            body = json.dumps(payload, separators=(",", ":")).encode("utf-8")
            headers["Content-Type"] = "application/json"
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        safe = method == "GET" or idempotency_key is not None
        attempt = 0
        while True:
            try:
                response = self.transport(method, self.base_url + path, headers, body, self.timeout)
            except ResponseBodyLimitExceeded as error:
                metadata = self._metadata(error.headers)
                raise CloverError(error.status, str(error), None, metadata) from error
            self.last_metadata = self._metadata(response.headers)
            if len(response.body) > self.max_response_body_bytes:
                raise CloverError(
                    response.status,
                    "Clover response body exceeds the configured limit",
                    None,
                    self.last_metadata,
                )
            decoded = self._decode(response.body)
            if 200 <= response.status < 300:
                return decoded
            if safe and response.status in self.RETRYABLE and attempt < self.max_retries:
                delay = self.last_metadata.get("retry_after")
                self.sleep(float(delay) if delay is not None else self.retry_base_delay * (2 ** attempt))
                attempt += 1
                continue
            problem = decoded if self._is_problem(decoded) else None
            message = str(problem.get("title")) if problem else f"Clover request failed ({response.status})"
            raise CloverError(response.status, message, problem, self.last_metadata)

    @staticmethod
    def _path_segment(value: str) -> str:
        if not isinstance(value, str) or not value:
            raise ValueError("email_id is required")
        return urllib.parse.quote(value, safe="")

    @staticmethod
    def _decode(body: bytes) -> dict[str, Any]:
        if not body:
            return {}
        try:
            value = json.loads(body.decode("utf-8"))
            return value if isinstance(value, dict) else {"data": value}
        except (UnicodeDecodeError, json.JSONDecodeError):
            return {"raw": body.decode("utf-8", errors="replace")}

    @staticmethod
    def _is_problem(value: Mapping[str, Any]) -> bool:
        return all(key in value for key in ("type", "title", "status", "code"))

    @staticmethod
    def _metadata(headers: Mapping[str, str]) -> dict[str, Any]:
        lowered = {key.lower(): value for key, value in headers.items()}
        result: dict[str, Any] = {
            "request_id": lowered.get("x-request-id"),
            "replayed": (lowered.get("idempotency-replayed") or "").lower() == "true",
            "rate_limit_remaining": None,
        }
        try:
            result["rate_limit_remaining"] = int(lowered["x-ratelimit-remaining"])
        except (KeyError, ValueError):
            pass
        try:
            retry = int(lowered["retry-after"])
            if retry >= 0:
                result["retry_after"] = retry
        except (KeyError, ValueError):
            pass
        return result
