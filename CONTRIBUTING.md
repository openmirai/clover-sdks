# Contributing

Keep changes scoped to one client unless the API contract requires a coordinated
conformance update. Run the corresponding `make check-<platform>` target and the
repository layout check before opening a pull request.

Do not hand-edit generated registry artifacts, commit build output, or add a
cross-language abstraction that bypasses native package conventions. Changes to
shared HTTP semantics must include equivalent conformance tests in every
affected maintained client.

Releases are tag-driven from `main`; package versions and changelogs must be
updated before creating the package-scoped tag documented in the root README.
