"""Cypress / Mocha output. Failures print as `N) suite test` and `✗ test`."""

from __future__ import annotations

import re

from lib.signatures import strip_ts

_MOCHA = re.compile(r"^\d+\)\s+(.*)$")
_CROSS = re.compile(r"^[✕✗×]\s+(.*)$")


def parse(log: str) -> list[dict]:
    tests: dict[str, str] = {}
    for raw in log.splitlines():
        line = strip_ts(raw).strip()
        m = _MOCHA.match(line) or _CROSS.match(line)
        if m:
            name = m.group(1).strip()
            if name:
                tests.setdefault(name, line)
    return [{"test_id": name, "error_line": err} for name, err in tests.items()]
