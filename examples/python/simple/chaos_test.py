# > Simple
import argparse
import asyncio
import signal
import threading
import time
import traceback
from typing import Any

from datetime import datetime, timezone
from pathlib import Path

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

FAILURE_LOG = Path(__file__).parent / "failures.log"

# Track the current worker so we can clean up on Ctrl+C
_current_worker = None
_current_thread = None
# poetry run python ./simple/worker_test.py --suffix new


def log_failure(phase: str, error: Exception) -> None:
    """Log a failure loudly to stderr and append to the failures log file."""
    timestamp = datetime.now(timezone.utc).isoformat()
    tb = traceback.format_exception(type(error), error, error.__traceback__)
    tb_str = "".join(tb)

    msg = f"[{timestamp}] FAILURE during {phase}: {error}\n{tb_str}"

    # Loud stderr output
    print(f"\n{'!' * 60}", flush=True)
    print(f"!!! FAILURE: {phase} !!!", flush=True)
    print(msg, flush=True)
    print(f"{'!' * 60}\n", flush=True)

    # Append to log file
    with open(FAILURE_LOG, "a") as f:
        f.write(msg)
        f.write("-" * 60 + "\n")


@hatchet.task()
def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:
    print("Executing simple task!")
    return {"result": "Hello, world!"}


@hatchet.durable_task()
def simple_durable(input: EmptyModel, ctx: Context) -> dict[str, str]:
    print("Executing durable task!")
    return {"result": "Hello from durable!"}


def _force_stop_worker(worker: Any, thread: threading.Thread) -> None:
    """Forcefully terminate the worker and its child processes."""
    worker.killing = True
    worker._terminate_processes()
    worker._close_queues()
    if worker.loop and worker.loop.is_running():
        worker.loop.call_soon_threadsafe(worker.loop.stop)
    thread.join(timeout=5)


def start_worker(suffix: str = "") -> tuple[Any, threading.Thread]:
    """Create and start a worker in a background thread."""
    name = f"test-worker-{suffix}" if suffix else "test-worker"
    worker = hatchet.worker(
        name,
        workflows=[simple, simple_durable],
        slots=10,
    )
    worker.handle_kill = False  # Prevent sys.exit on shutdown

    # Restore default signal handlers so Ctrl+C raises KeyboardInterrupt
    signal.signal(signal.SIGINT, signal.default_int_handler)
    signal.signal(signal.SIGTERM, signal.SIG_DFL)

    thread = threading.Thread(target=worker.start, daemon=True)
    thread.start()

    # Give the worker a moment to initialize
    time.sleep(2)
    print("Worker connected.")
    return worker, thread


def stop_worker(worker: Any, thread: threading.Thread) -> None:
    """Stop the worker gracefully."""
    try:
        if worker.loop and worker.loop.is_running():
            asyncio.run_coroutine_threadsafe(worker.exit_gracefully(), worker.loop)
        thread.join(timeout=10)
        if thread.is_alive():
            _force_stop_worker(worker, thread)
        print("Worker disconnected.")
    except Exception as e:
        log_failure("worker disconnect", e)


def main() -> None:
    global _current_worker, _current_thread

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--suffix",
        default="",
        help="Suffix to append to the worker name (e.g. 'old' or 'new')",
    )
    args = parser.parse_args()

    try:
        while True:
            # --- Connect the worker ---
            print("\n=== Connecting worker ===")
            try:
                worker, thread = start_worker(args.suffix)
                _current_worker, _current_thread = worker, thread
            except Exception as e:
                log_failure("worker connect", e)
                time.sleep(5)
                continue

            # --- Trigger tasks every 1 second for 5 seconds ---
            for tick in range(5):
                time.sleep(1)
                print(f"\n--- Triggering tasks (tick {tick + 1}/5) ---")
                try:
                    ref = simple.run_no_wait()
                    print(f"Task triggered: {ref}")
                except Exception as e:
                    log_failure(f"task trigger (tick {tick + 1}/5)", e)
                try:
                    ref = simple_durable.run_no_wait()
                    print(f"Durable task triggered: {ref}")
                except Exception as e:
                    log_failure(f"durable task trigger (tick {tick + 1}/5)", e)

            # --- Disconnect the worker ---
            print("\n=== Disconnecting worker ===")
            stop_worker(worker, thread)
            _current_worker, _current_thread = None, None

    except KeyboardInterrupt:
        print("\n\nCtrl+C received, shutting down...")
        if _current_worker and _current_thread:
            _force_stop_worker(_current_worker, _current_thread)
        print("Bye!")



if __name__ == "__main__":
    main()
