# clover-sdk

Dependency-free Python 3.10+ client for Clover's email API. `CloverClient`
uses `urllib`, sends bearer authentication and a `User-Agent`, and requires an
`idempotency_key` for `send`, `send_batch`, `schedule`, and `cancel`.

```python
from clover_sdk import CloverClient

client = CloverClient("https://api.example.com", "re_public_secret")
accepted = client.send(
    {
        "from": {"address": "sender@example.com"},
        "to": [{"address": "user@example.com"}],
        "subject": "Hello",
        "text": "Accepted asynchronously",
    },
    "order-1234",
)
```

The client also exposes `get`, cursor-aware `list`, `send_batch`, `schedule`,
and `cancel`. `CloverError.problem` preserves unknown JSON members from
`application/problem+json`; `last_metadata` exposes request ID, replay,
rate-limit, and retry-after headers. GETs and idempotent mutations retry only
transient statuses, with a maximum of three retries. Inject `transport` and
`sleep` for deterministic tests.

Development checks use `pytest`, `ruff`, and `mypy`:

```bash
python -m pip install -e '.[dev]'
python -m ruff check .
python -m ruff format --check .
python -m mypy
python -m pytest --cov
```

See [`RELEASING.md`](RELEASING.md) for the build/twine publication path and
[`SECURITY.md`](SECURITY.md) for vulnerability reporting.
