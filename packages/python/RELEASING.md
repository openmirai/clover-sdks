# Releasing

Run the complete local check set (`ruff`, `mypy`, `pytest --cov`, `build`, and
`twine check`) from a clean checkout. Update `CHANGELOG.md`, create a signed
`python/vX.Y.Z` tag, and push it. The release workflow builds an sdist and wheel and
publishes through PyPI trusted publishing; it never runs for pull requests.
