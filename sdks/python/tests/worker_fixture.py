import logging
import os
import subprocess
import time
from io import BytesIO
from threading import Thread
from typing import Callable, Generator

import psutil
import requests


def wait_for_worker_health() -> bool:
    worker_healthcheck_attempts = 0
    max_healthcheck_attempts = 25

    while True:
        if worker_healthcheck_attempts > max_healthcheck_attempts:
            raise Exception(
                f"Worker failed to start within {max_healthcheck_attempts} seconds"
            )

        try:
            requests.get("http://localhost:8001/health", timeout=5)

            return True
        except Exception:
            time.sleep(1)

        worker_healthcheck_attempts += 1


def log_output(pipe: BytesIO, log_func: Callable[[str], None]) -> None:
    for line in iter(pipe.readline, b""):
        print(line.decode().strip())


def hatchet_worker(
    command: list[str],
) -> Generator[subprocess.Popen[bytes], None, None]:
    logging.info(f"Starting background worker: {' '.join(command)}")

    proc = subprocess.Popen(
        command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=os.environ.copy()
    )

    # Check if the process is still running
    if proc.poll() is not None:
        raise Exception(f"Worker failed to start with return code {proc.returncode}")

    Thread(target=log_output, args=(proc.stdout, logging.info), daemon=True).start()
    Thread(target=log_output, args=(proc.stderr, logging.error), daemon=True).start()

    wait_for_worker_health()

    yield proc

    logging.info("Cleaning up background worker")

    parent = psutil.Process(proc.pid)
    children = parent.children(recursive=True)

    for child in children:
        child.terminate()

    parent.terminate()

    _, alive = psutil.wait_procs([parent] + children, timeout=5)

    for p in alive:
        logging.warning(f"Force killing process {p.pid}")
        p.kill()
