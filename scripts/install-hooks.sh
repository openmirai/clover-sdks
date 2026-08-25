#!/bin/sh
set -eu

if ! command -v pre-commit >/dev/null 2>&1; then
  echo "pre-commit 4.6.2 is required; install it with: python3 -m pip install pre-commit==4.6.2" >&2
  exit 1
fi

if [ "$(pre-commit --version)" != "pre-commit 4.6.2" ]; then
  echo "pre-commit 4.6.2 is required; found: $(pre-commit --version)" >&2
  exit 1
fi

global_hooks_path="$(git config --global --get core.hooksPath || true)"
if [ -n "$global_hooks_path" ]; then
  GIT_CONFIG_GLOBAL=/dev/null pre-commit install --install-hooks --hook-type pre-commit --hook-type commit-msg
else
  pre-commit install --install-hooks --hook-type pre-commit --hook-type commit-msg
fi

pre-commit validate-config .pre-commit-config.yaml
