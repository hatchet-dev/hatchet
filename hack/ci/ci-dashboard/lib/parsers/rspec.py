"""RSpec output. The rerun summary lists each failure as
`rspec ./spec/foo_spec.rb:12 # description`."""

from __future__ import annotations

import re

from lib.signatures import strip_ts

_RSPEC = re.compile(r"^rspec\s+(\S+:\d+)\s+#\s+(.*)$")


def parse(log: str) -> list[dict]:
    tests: dict[str, str] = {}
    for raw in log.splitlines():
        line = strip_ts(raw).strip()
        m = _RSPEC.match(line)
        if m:
            tests.setdefault(m.group(1), f"{m.group(1)} # {m.group(2)}")
    return [{"test_id": name, "error_line": err} for name, err in tests.items()]
