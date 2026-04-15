from collections.abc import AsyncGenerator, Generator
from subprocess import Popen
from typing import cast

import pytest
import pytest_asyncio
from pytest import FixtureRequest

from hatchet_sdk import Hatchet
from hatchet_sdk.deprecated.deprecation import semver_less_than
from hatchet_sdk.engine_version import MinEngineVersion
from tests.worker_fixture import get_free_port, hatchet_worker


@pytest_asyncio.fixture(scope="session", loop_scope="session")
async def hatchet() -> AsyncGenerator[Hatchet, None]:
    yield Hatchet()


@pytest_asyncio.fixture(scope="session", loop_scope="session")
async def engine_version(hatchet: Hatchet) -> str | None:
    return await hatchet.aio_get_engine_version()


@pytest_asyncio.fixture(scope="session", loop_scope="session")
async def supports_durable_eviction(engine_version: str | None) -> bool:
    if not engine_version:
        return False
    return not semver_less_than(engine_version, MinEngineVersion.DURABLE_EVICTION)


@pytest.fixture()
def _skip_unless_durable_eviction(supports_durable_eviction: bool) -> None:
    if not supports_durable_eviction:
        pytest.skip(
            f"Engine does not support durable eviction (requires >= {MinEngineVersion.DURABLE_EVICTION})"
        )


@pytest_asyncio.fixture(scope="session", loop_scope="session")
async def supports_observability(engine_version: str | None) -> bool:
    if not engine_version:
        return False
    return not semver_less_than(engine_version, MinEngineVersion.OBSERVABILITY)


@pytest.fixture()
def _skip_unless_observability(supports_observability: bool) -> None:
    if not supports_observability:
        pytest.skip(
            f"Engine does not support observability (requires >= {MinEngineVersion.OBSERVABILITY})"
        )


@pytest.fixture(scope="session", autouse=True)
def worker() -> Generator[Popen[bytes], None, None]:
    command = ["poetry", "run", "python", "examples/worker.py"]

    with hatchet_worker(command) as proc:
        yield proc


def _on_demand_worker_fixture(
    request: FixtureRequest,
) -> Generator[Popen[bytes], None, None]:
    command = cast(list[str], request.param)

    with hatchet_worker(command, get_free_port()) as proc:
        yield proc


@pytest.fixture(scope="session")
def on_demand_worker(request: FixtureRequest) -> Generator[Popen[bytes], None, None]:
    yield from _on_demand_worker_fixture(request)
