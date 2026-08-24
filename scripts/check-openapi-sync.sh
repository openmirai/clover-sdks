#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
backend_swagger="${repo_root}/../clover/backend/cmd/api/docs/swagger.json"
sdk_openapi="${repo_root}/openapi/clover-v1.json"

if [[ ! -f "${backend_swagger}" ]]; then
  echo "backend Swagger not found: ${backend_swagger}" >&2
  exit 1
fi
if [[ ! -f "${sdk_openapi}" ]]; then
  echo "SDK OpenAPI snapshot not found: ${sdk_openapi}" >&2
  exit 1
fi

if ! cmp -s "${backend_swagger}" "${sdk_openapi}"; then
  echo "OpenAPI snapshot is out of sync with backend Swagger" >&2
  echo "sync: cp ../clover/backend/cmd/api/docs/swagger.json openapi/clover-v1.json" >&2
  exit 1
fi

echo "OpenAPI snapshot is synchronized"
