"""pytest output. Failures show up in the short summary as
`FAILED path::test - SomeError: msg` and/or inline as `path::test FAILED`."""

from __future__ import annotations

import re

from lib.signatures import strip_ts

_SUMMARY = re.compile(r"^(?:FAILED|ERROR)\s+(\S+::\S+)")
_INLINE = re.compile(r"^(\S+::\S+)\s+(?:FAILED|ERROR)\b")


def parse(log: str) -> list[dict]:
    tests: dict[str, str] = {}
    for raw in log.splitlines():
        line = strip_ts(raw).strip()
        m = _SUMMARY.match(line) or _INLINE.match(line)
        if m:
            tests.setdefault(m.group(1), line)
    return [{"test_id": name, "error_line": err} for name, err in tests.items()]
