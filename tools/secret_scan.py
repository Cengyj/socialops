#!/usr/bin/env python3
"""High-confidence secret scanner for local verification.

The project intentionally keeps config examples, OAuth field names, and test
tokens in source. This scanner therefore reports only token shapes that are
unlikely to be harmless placeholders.
"""

from __future__ import annotations

import os
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]

SKIP_DIR_NAMES = {
    ".codex-artifacts",
    ".codex-logs",
    ".codex-qa",
    ".codex-screenshots",
    ".cache",
    ".git",
    ".idea",
    ".omx",
    ".pnpm-store",
    ".pytest_cache",
    ".venv",
    "dist",
    "artifacts",
    "codex-screenshots",
    "node_modules",
    "qa-artifacts",
    "qa-screenshots",
}

SKIP_PATHS = {
    ".cache",
    "backend/.cache",
    "backend/internal/web/dist",
    "backend/socialops",
    "deploy/postgres_data",
    "deploy/redis_data",
}

SKIP_EXTS = {
    ".7z",
    ".avif",
    ".bin",
    ".db",
    ".gif",
    ".gz",
    ".ico",
    ".jpeg",
    ".jpg",
    ".lock",
    ".pdf",
    ".png",
    ".sqlite",
    ".tar",
    ".webp",
    ".zip",
}

PRIVATE_KEY_BLOCK = re.compile(
    r"-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----"
    r"(?P<body>.*?)"
    r"-----END (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----",
    re.DOTALL,
)

PATTERNS = [
    ("aws-access-key", re.compile(r"\bAKIA[0-9A-Z]{16}\b")),
    ("github-token", re.compile(r"\bgh[pousr]_[A-Za-z0-9_]{36,255}\b")),
    ("openai-key", re.compile(r"\bsk-(?:proj-)?[A-Za-z0-9_-]{32,}\b")),
    ("slack-token", re.compile(r"\bxox[baprs]-[A-Za-z0-9-]{20,}\b")),
    ("stripe-live-secret", re.compile(r"\bsk_live_[A-Za-z0-9]{20,}\b")),
]


def is_skipped(path: Path) -> bool:
    rel = path.relative_to(ROOT).as_posix()
    parts = rel.split("/")
    if any(part in SKIP_DIR_NAMES for part in parts):
        return True
    for idx in range(len(parts)):
        if "/".join(parts[: idx + 1]) in SKIP_PATHS:
            return True
    return path.suffix.lower() in SKIP_EXTS


def iter_files() -> list[Path]:
    files: list[Path] = []
    for dirpath, dirnames, filenames in os.walk(ROOT):
        current = Path(dirpath)
        dirnames[:] = [
            name
            for name in dirnames
            if not is_skipped(current / name)
        ]
        for filename in filenames:
            path = current / filename
            if not is_skipped(path):
                files.append(path)
    return files


def scan_file(path: Path) -> list[tuple[str, int]]:
    try:
        content = path.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return []
    except OSError as exc:
        print(f"warning: cannot read {path.relative_to(ROOT)}: {exc}", file=sys.stderr)
        return []

    findings: list[tuple[str, int]] = []
    for match in PRIVATE_KEY_BLOCK.finditer(content):
        body = match.group("body").strip()
        if "..." not in body and body != "data" and len(body) >= 128:
            line_no = content[: match.start()].count("\n") + 1
            findings.append(("private-key", line_no))

    for line_no, line in enumerate(content.splitlines(), start=1):
        for name, pattern in PATTERNS:
            if pattern.search(line):
                findings.append((name, line_no))
    return findings


def main() -> int:
    findings: list[tuple[Path, str, int]] = []
    for path in iter_files():
        for name, line_no in scan_file(path):
            findings.append((path, name, line_no))

    if findings:
        print("Potential secrets found:")
        for path, name, line_no in findings:
            print(f"- {path.relative_to(ROOT)}:{line_no} [{name}]")
        return 1

    print("Secret scan passed: no high-confidence secrets found.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
