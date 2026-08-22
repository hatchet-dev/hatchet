"""Go `go test` output. Failing tests appear as `--- FAIL: TestName (12.34s)`,
including indented subtests like `    --- FAIL: TestName/sub (0.01s)`."""

from __future__ import annotations

import re

from lib.signatures import strip_ts

_FAIL = re.compile(r"^--- FAIL: (\S+)")


def parse(log: str) -> list[dict]:
    tests: dict[str, str] = {}
    for raw in log.splitlines():
        line = strip_ts(raw).strip()
        m = _FAIL.match(line)
        if m:
            tests.setdefault(m.group(1), line)
    return [{"test_id": name, "error_line": err} for name, err in tests.items()]
