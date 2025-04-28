from subprocess import Popen
from typing import AsyncGenerator, Generator, cast

import pytest
import pytest_asyncio
from pytest import FixtureRequest

from hatchet_sdk import Hatchet
from tests.worker_fixture import hatchet_worker


@pytest_asyncio.fixture(scope="session", loop_scope="session")
async def hatchet() -> AsyncGenerator[Hatchet, None]:
    yield Hatchet(
        debug=True,
    )


@pytest.fixture(scope="session", autouse=True)
def worker() -> Generator[Popen[bytes], None, None]:
    command = ["poetry", "run", "python", "examples/worker.py"]

    with hatchet_worker(command) as proc:
        yield proc


@pytest.fixture(scope="session")
def on_demand_worker(request: FixtureRequest) -> Generator[Popen[bytes], None, None]:
    command, port = cast(tuple[list[str], int], request.param)

    with hatchet_worker(command, port) as proc:
        yield proc
