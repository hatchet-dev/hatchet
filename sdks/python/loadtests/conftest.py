from collections.abc import AsyncGenerator, Generator
from subprocess import Popen

import psutil
import pytest
import pytest_asyncio

from hatchet_sdk import Hatchet
from tests.worker_fixture import hatchet_worker


@pytest_asyncio.fixture(scope="session", loop_scope="session")
async def hatchet() -> AsyncGenerator[Hatchet, None]:
    yield Hatchet(debug=True)


@pytest.fixture(scope="session", autouse=True)
def worker() -> Generator[Popen[bytes], None, None]:
    command = ["poetry", "run", "python", "-m", "loadtests.worker"]
    with hatchet_worker(command, healthcheck_port=8005) as proc:
        yield proc


@pytest.fixture(scope="session", autouse=True)
def memory_leak_check(request: pytest.FixtureRequest, worker: Popen[bytes]) -> None:
    """Fail if worker RSS grows too much between start and end of load tests."""
    proc = psutil.Process(worker.pid)
    initial_rss_mb = proc.memory_info().rss / 1024 / 1024

    def check_at_end() -> None:
        try:
            final_rss_mb = proc.memory_info().rss / 1024 / 1024
            delta_mb = final_rss_mb - initial_rss_mb
            max_growth_mb = 50
            if delta_mb > max_growth_mb:
                pytest.fail(
                    f"Possible memory leak: worker RSS grew {delta_mb:.1f} MB "
                    f"(from {initial_rss_mb:.1f} to {final_rss_mb:.1f} MB). "
                    f"Threshold: {max_growth_mb} MB."
                )
        except psutil.NoSuchProcess:
            pass

    request.addfinalizer(check_at_end)
