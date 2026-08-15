# Contributing

Use Node 24 LTS (the CI matrix also verifies Node 20 and 22). Install with
`npm install`, enable hooks with `pre-commit install`, and run `npm test`,
`npm run lint`, and `npm run format:check` before opening a pull request.

Keep public API changes documented in the README and `CHANGELOG.md`; do not
commit generated `dist/` or coverage output.
