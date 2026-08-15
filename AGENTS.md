# Repository guidance

This is a polyglot SDK monorepo. Preserve each package's public API and native
tooling. Shared behavior changes require conformance coverage in every affected
maintained SDK. Run the package-specific `make check-*` target plus
`make check-layout` before handoff.

Do not add implementations for documentation-only community targets without an
explicit maintenance decision. Do not commit generated build output, dependency
caches, registry credentials, or release artifacts.
