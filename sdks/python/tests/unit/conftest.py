"""Override the session-scoped autouse worker fixture from the root conftest
so that pure unit tests can run without a live Hatchet server."""

from __future__ import annotations

from collections.abc import Iterator

import pytest


@pytest.fixture(scope="session", autouse=True)
def worker() -> Iterator[None]:
    yield None
