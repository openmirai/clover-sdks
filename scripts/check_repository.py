from __future__ import annotations

from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]

REQUIRED = (
    "Package.swift",
    "openapi/clover-v1.json",
    "apps/cli/go.mod",
    "packages/dart/pubspec.yaml",
    "packages/go/go.mod",
    "packages/java/pom.xml",
    "packages/python/pyproject.toml",
    "packages/rust/Cargo.toml",
    "packages/swift/Sources/CloverSDK/CloverClient.swift",
    "packages/typescript/package.json",
)

for relative in REQUIRED:
    if not (ROOT / relative).is_file():
        raise SystemExit(f"missing monorepo package file: {relative}")

nested_workflows = [
    path.relative_to(ROOT)
    for path in ROOT.glob("**/.github/workflows/*")
    if path.parent.parent.parent != ROOT
]
if nested_workflows:
    raise SystemExit(f"nested GitHub workflows are not active: {nested_workflows}")

legacy_repositories: list[Path] = []
legacy_prefixes = (
    "github.com/openmirai/" + "clover-sdk-",
    "github.com/openmirai/" + "clover-cli",
)
for path in ROOT.rglob("*"):
    if not path.is_file() or ".git" in path.parts:
        continue
    try:
        content = path.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError):
        continue
    if any(prefix in content for prefix in legacy_prefixes):
        legacy_repositories.append(path.relative_to(ROOT))

if legacy_repositories:
    raise SystemExit(f"standalone repository references remain: {legacy_repositories}")

print("monorepo layout check passed")
