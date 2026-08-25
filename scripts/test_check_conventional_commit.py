from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from check_conventional_commit import is_conventional_subject, read_subject


class ConventionalCommitTest(unittest.TestCase):
    def test_accepts_conventional_release_subject(self) -> None:
        self.assertTrue(
            is_conventional_subject("chore(release): publish @sendclover/sdk v0.1.0")
        )

    def test_accepts_scope_breaking_change_and_autosquash(self) -> None:
        self.assertTrue(is_conventional_subject("feat(sdk)!: require environment scope"))
        self.assertTrue(is_conventional_subject("fixup! fix(transport): reject malformed envelopes"))

    def test_accepts_git_merge_subject(self) -> None:
        self.assertTrue(is_conventional_subject("Merge branch 'main' into release"))

    def test_rejects_non_conventional_subjects(self) -> None:
        for subject in ("release 0.1.0", "Feat: uppercase type", "fix missing separator"):
            with self.subTest(subject=subject):
                self.assertFalse(is_conventional_subject(subject))

    def test_reads_first_non_comment_subject(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            message = Path(directory, "COMMIT_EDITMSG")
            message.write_text(
                "\n# template hint\n\nfix(sdk): preserve scope\n\nbody\n", encoding="utf-8"
            )
            self.assertEqual(read_subject(message), "fix(sdk): preserve scope")


if __name__ == "__main__":
    unittest.main()
