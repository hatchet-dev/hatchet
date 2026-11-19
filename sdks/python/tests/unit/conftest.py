"""Override the session-scoped autouse worker fixture from the root conftest
so that pure unit tests can run without a live Hatchet server."""

import pytest


@pytest.fixture(scope="session", autouse=True)
def worker() -> None:
    return None
