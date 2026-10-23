from collections.abc import AsyncGenerator, Generator
from contextlib import AbstractContextManager, nullcontext
from subprocess import Popen
from typing import Any, TypeAlias, TypedDict, cast

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


@pytest_asyncio.fixture(scope="session", loop_scope="session")
async def supports_batching(engine_version: str | None) -> bool:
    if not engine_version:
        return False
    return not semver_less_than(engine_version, MinEngineVersion.BATCHING)


@pytest.fixture()
def _skip_unless_batching(supports_batching: bool) -> None:
    if not supports_batching:
        pytest.skip(
            f"Engine does not support batch tasks (requires >= {MinEngineVersion.BATCHING})"
        )


@pytest.fixture(scope="session", autouse=True)
def worker() -> Generator[Popen[bytes], None, None]:
    command = ["poetry", "run", "python", "examples/worker.py"]

    with hatchet_worker(command, get_free_port()) as proc:
        yield proc


def _on_demand_worker_fixture(
    request: FixtureRequest,
    print_logs: bool = True,
) -> Generator[Popen[bytes], None, None]:
    if isinstance(request.param, dict):
        param_dict = cast(OnDemandWorkerFixtureRequestDict, request.param)
        command = param_dict["cmd"]
    else:
        command = cast(list[str], request.param)

    with hatchet_worker(command, get_free_port(), print_logs) as proc:
        yield proc


class OnDemandWorkerFixtureRequestDict(TypedDict):
    cmd: list[str]
    caplog: str | None


OnDemandWorkerFixtureRequest: TypeAlias = list[str] | OnDemandWorkerFixtureRequestDict


@pytest.fixture()
def on_demand_worker(
    request: FixtureRequest, caplog: pytest.LogCaptureFixture
) -> Generator[Popen[bytes], None, None]:
    # Default to print worker logs, else use caplog at specified level
    caplog_ctx: AbstractContextManager[Any, Any] = nullcontext()
    print_logs = True
    if isinstance(request.param, dict):
        param_dict = cast(OnDemandWorkerFixtureRequestDict, request.param)
        if (caplog_level := param_dict["caplog"]) is not None:
            caplog_ctx = caplog.at_level(caplog_level)
            print_logs = False
    with caplog_ctx:
        yield from _on_demand_worker_fixture(request, print_logs=print_logs)
