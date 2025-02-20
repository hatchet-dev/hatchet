import logging
import subprocess
import time
from io import BytesIO
from threading import Thread
from typing import AsyncGenerator, Callable, cast

import psutil
import pytest
import pytest_asyncio

from hatchet_sdk import Hatchet


@pytest_asyncio.fixture(scope="session")
async def aiohatchet() -> AsyncGenerator[Hatchet, None]:
    yield Hatchet(debug=True)


@pytest.fixture(scope="session")
def hatchet() -> Hatchet:
    return Hatchet(debug=True)


@pytest.fixture()
def worker(request: pytest.FixtureRequest):
    example = cast(str, request.param)

    command = ["poetry", "run", example]

    logging.info(f"Starting background worker: {' '.join(command)}")
    proc = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

    # Check if the process is still running
    if proc.poll() is not None:
        raise Exception(f"Worker failed to start with return code {proc.returncode}")

    time.sleep(5)

    def log_output(pipe: BytesIO, log_func: Callable[[str], None]) -> None:
        for line in iter(pipe.readline, b""):
            log_func(line.decode().strip())

    Thread(target=log_output, args=(proc.stdout, logging.info), daemon=True).start()
    Thread(target=log_output, args=(proc.stderr, logging.error), daemon=True).start()

    yield proc

    logging.info("Cleaning up background worker")
    parent = psutil.Process(proc.pid)
    children = parent.children(recursive=True)
    for child in children:
        child.terminate()
    parent.terminate()

    _, alive = psutil.wait_procs([parent] + children, timeout=3)
    for p in alive:
        logging.warning(f"Force killing process {p.pid}")
        p.kill()
