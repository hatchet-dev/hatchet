"""Jest / Vitest output. Failing tests print as `✕ test name (12 ms)` or
`● Suite › test`; failing files as `FAIL src/foo.spec.ts`."""

from __future__ import annotations

import re

from lib.signatures import strip_ts

_CROSS = re.compile(r"^[✕✗×]\s+(.*?)(?:\s+\(\d+\s*m?s\))?$")
_BULLET = re.compile(r"^[●•]\s+(.*)$")
_FAILFILE = re.compile(r"^FAIL\s+(\S+)")


def parse(log: str) -> list[dict]:
    tests: dict[str, str] = {}
    files: dict[str, str] = {}
    for raw in log.splitlines():
        line = strip_ts(raw).strip()
        m = _CROSS.match(line) or _BULLET.match(line)
        if m:
            name = m.group(1).strip()
            if name:
                tests.setdefault(name, line)
            continue
        mf = _FAILFILE.match(line)
        if mf:
            files.setdefault(mf.group(1), line)
    if tests:
        return [{"test_id": name, "error_line": err} for name, err in tests.items()]
    return [{"test_id": name, "error_line": err} for name, err in files.items()]
