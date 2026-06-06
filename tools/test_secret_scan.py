import unittest

from tools import secret_scan


class SecretScanSkipTest(unittest.TestCase):
    def test_skips_generated_local_artifact_directories(self) -> None:
        generated_dirs = [
            ".codex-artifacts",
            ".codex-logs",
            ".codex-qa",
            ".codex-screenshots",
            "artifacts",
            "codex-screenshots",
            "qa-artifacts",
            "qa-screenshots",
        ]

        for rel_path in generated_dirs:
            with self.subTest(rel_path=rel_path):
                self.assertTrue(secret_scan.is_skipped(secret_scan.ROOT / rel_path))


if __name__ == "__main__":
    unittest.main()
