from collections.abc import AsyncGenerator, Generator
from subprocess import Popen

import pytest
import pytest_asyncio

from hatchet_sdk import Hatchet
from tests.worker_fixture import hatchet_worker


@pytest_asyncio.fixture(scope="session", loop_scope="session")
async def hatchet() -> AsyncGenerator[Hatchet, None]:
    yield Hatchet(debug=True)


@pytest.fixture(scope="session", autouse=True)
def worker() -> Generator[Popen[bytes], None, None]:
    command = ["poetry", "run", "python", "-m", "examples.durable_eviction.worker"]
    with hatchet_worker(command, healthcheck_port=8003) as proc:
        yield proc
