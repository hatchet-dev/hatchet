"""Fallback parser: no per-test breakdown, just the first error-like line so the
failure still gets a stable signature."""

from __future__ import annotations

import re

from lib.signatures import strip_ts

_ERRISH = re.compile(
    r"(?i)\b(error|failed|failure|panic|fatal|exception|timed out|timeout|cannot|no such)\b"
)
_NOISE = re.compile(r"(?i)(##\[group\]|##\[endgroup\]|deprecat|warning: node)")


def parse(log: str) -> list[dict]:
    for raw in log.splitlines():
        line = strip_ts(raw).strip()
        if not line or _NOISE.search(line):
            continue
        if _ERRISH.search(line):
            return [{"test_id": None, "error_line": line}]
    return []
