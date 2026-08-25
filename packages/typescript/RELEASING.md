# Releasing

1. Update `CHANGELOG.md` and `package.json`, then run `npm run quality` and
   `npm pack --dry-run` from `packages/typescript`.
2. From a clean checkout, push the signed `typescript/vX.Y.Z` tag. The tag
   workflow performs the package quality gate and publishes with npm Trusted
   Publishing (OIDC) and public access; no long-lived npm token or local
   release helper is required. npm provenance is unavailable while the source
   repository remains private.
3. Verify the package page, tarball contents, and generated GitHub release
   before announcing the version.

The release workflow is tag-triggered and must not receive long-lived secrets
in pull requests.
