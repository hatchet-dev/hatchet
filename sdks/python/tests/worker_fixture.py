import contextlib
import logging
import os
import socket
import subprocess
import time
from collections.abc import Callable, Generator
from contextlib import contextmanager
from io import BytesIO
from threading import Thread
from typing import cast

import psutil
import requests


def get_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("", 0))
        return cast(int, s.getsockname()[1])


def wait_for_worker_health(healthcheck_port: int) -> bool:
    worker_healthcheck_attempts = 0
    max_healthcheck_attempts = 25

    while True:
        if worker_healthcheck_attempts > max_healthcheck_attempts:
            raise Exception(
                f"Worker failed to start within {max_healthcheck_attempts} seconds"
            )

        try:
            resp = requests.get(
                f"http://localhost:{healthcheck_port}/health",
                timeout=5,
            )

            if resp.status_code == 200:
                return True

            time.sleep(1)
        except Exception:
            time.sleep(1)

        worker_healthcheck_attempts += 1


def log_output(pipe: BytesIO, log_func: Callable[[str], None]) -> None:
    for line in iter(pipe.readline, b""):
        print(line.decode().strip())


@contextmanager
def hatchet_worker(
    command: list[str],
    healthcheck_port: int = 8001,
) -> Generator[subprocess.Popen[bytes], None, None]:
    logging.info(f"Starting background worker: {' '.join(command)}")

    os.environ["HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT"] = str(healthcheck_port)
    env = os.environ.copy()

    proc = subprocess.Popen(
        command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env
    )

    # Check if the process is still running
    if proc.poll() is not None:
        raise Exception(f"Worker failed to start with return code {proc.returncode}")

    Thread(target=log_output, args=(proc.stdout, logging.info), daemon=True).start()
    Thread(target=log_output, args=(proc.stderr, logging.error), daemon=True).start()

    # The finally block also runs when the health check raises, so a worker
    # that never came up is still torn down instead of leaking.
    try:
        wait_for_worker_health(healthcheck_port=healthcheck_port)

        yield proc
    finally:
        logging.info("Cleaning up background worker")
        _terminate_worker(proc)


def _terminate_worker(proc: subprocess.Popen[bytes]) -> None:
    """Terminate the worker and its descendants, force killing anything that lingers.

    :param proc: The worker process handle owned by the fixture.
    """
    try:
        children = psutil.Process(proc.pid).children(recursive=True)
    except psutil.Error:
        # The worker already exited (a test may have signalled it itself), so
        # there is nothing left to enumerate under it.
        children = []

    for child in children:
        with contextlib.suppress(psutil.Error):
            child.terminate()

    with contextlib.suppress(ProcessLookupError):
        proc.terminate()

    # Wait on the Popen handle we own: that reaps our child with waitpid, which
    # cannot hit the pidfd_open() EINVAL race psutil.wait_procs runs into on
    # Linux 6.x when the worker has already exited (psutil issue 2715).
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        logging.warning("Force killing worker %s", proc.pid)
        proc.kill()
        proc.wait(timeout=5)

    _reap_descendants(children, timeout=5)


def _reap_descendants(children: list[psutil.Process], timeout: float) -> None:
    """Wait for the worker's descendants, force killing any that outlive the timeout.

    :param children: Descendant processes that were already sent SIGTERM.
    :param timeout: Total seconds to wait across all descendants.
    """
    deadline = time.monotonic() + timeout

    for child in children:
        remaining = max(0.0, deadline - time.monotonic())

        try:
            child.wait(timeout=remaining)
        except psutil.TimeoutExpired:
            logging.warning("Force killing process %s", child.pid)
            with contextlib.suppress(psutil.Error):
                child.kill()
        except (psutil.Error, OSError):
            # NoSuchProcess, or pidfd_open() failing on a process that already
            # exited and was reaped by init: either way it is gone.
            pass
