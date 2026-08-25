from __future__ import annotations

import re
import sys
from pathlib import Path


CONVENTIONAL_SUBJECT = re.compile(
    r"^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)"
    r"(?:\([a-z0-9][a-z0-9._/-]*\))?!?: \S.*$"
)
AUTOSQUASH_PREFIXES = ("fixup! ", "squash! ")


def is_conventional_subject(subject: str) -> bool:
    if subject.startswith("Merge "):
        return True
    for prefix in AUTOSQUASH_PREFIXES:
        if subject.startswith(prefix):
            return is_conventional_subject(subject.removeprefix(prefix))
    return CONVENTIONAL_SUBJECT.fullmatch(subject) is not None


def read_subject(message_path: Path) -> str:
    for line in message_path.read_text(encoding="utf-8").splitlines():
        subject = line.strip()
        if subject and not subject.startswith("#"):
            return subject
    return ""


def main(argv: list[str]) -> int:
    if len(argv) != 2:
        print("usage: check_conventional_commit.py <commit-message-file>", file=sys.stderr)
        return 2

    subject = read_subject(Path(argv[1]))
    if is_conventional_subject(subject):
        return 0

    print(
        "invalid commit subject; expected Conventional Commits, for example "
        "'chore(release): publish @sendclover/sdk v0.1.0'",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
