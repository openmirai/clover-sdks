# Releasing

1. Update `CHANGELOG.md`, run `npm run lint`, `npm run format:check`,
   `npm run typecheck`, and `npm run test:coverage`.
2. From a clean checkout, run `npm ci`, then `npm run release` locally with
   npm/GitHub credentials. `release-it` bumps the version and creates the
   release commit/tag; commit the changelog update and push the signed
   `typescript/vX.Y.Z` tag.
3. Configure npm trusted publishing for this repository/workflow (OIDC; no
   long-lived npm token) before pushing the tag. The tag workflow validates the
   SemVer tag, reruns `release-it` in its publish-only config (no duplicate
   tag/commit), publishes with npm provenance, and creates the GitHub release.
   Verify package contents before announcing it.

The release workflow is tag-triggered and must not receive long-lived secrets
in pull requests.
