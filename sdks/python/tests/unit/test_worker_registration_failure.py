import asyncio
import multiprocessing
from typing import cast
from unittest.mock import AsyncMock, MagicMock, patch

import grpc
import pytest

from hatchet_sdk.worker.action_listener_process import (
    WorkerActionListenerProcess,
    worker_action_listener_process,
)
from hatchet_sdk.worker.worker import Worker, WorkerStatus

LISTENER_MODULE = "hatchet_sdk.worker.action_listener_process"
WORKER_MODULE = "hatchet_sdk.worker.worker"


def _registration_error(
    status: grpc.StatusCode = grpc.StatusCode.INTERNAL,
) -> grpc.aio.AioRpcError:
    return grpc.aio.AioRpcError(
        status,
        initial_metadata=grpc.aio.Metadata(),
        trailing_metadata=grpc.aio.Metadata(),
        details="registration failed",
    )


@pytest.mark.parametrize(
    "status",
    [
        grpc.StatusCode.INTERNAL,
        grpc.StatusCode.UNAVAILABLE,
        grpc.StatusCode.UNAUTHENTICATED,
    ],
)
async def test_registration_failure_propagates_without_starting_tasks(
    status: grpc.StatusCode,
) -> None:
    config = MagicMock()
    config.healthcheck.enabled = False
    with patch(f"{LISTENER_MODULE}.Client"):
        process = WorkerActionListenerProcess(
            name="test-worker",
            actions=["test-action"],
            slot_config={"default": 1},
            config=config,
            action_queue=MagicMock(),
            event_queue=MagicMock(),
            handle_kill=False,
            debug=False,
            labels=[],
            worker_id_queue=MagicMock(),
            stop_event=MagicMock(),
        )

    error = _registration_error(status)
    dispatcher = MagicMock()
    dispatcher.get_action_listener = AsyncMock(side_effect=error)
    with patch(f"{LISTENER_MODULE}.DispatcherClient", return_value=dispatcher):
        with pytest.raises(grpc.aio.AioRpcError) as caught:
            await process.start()

    assert caught.value is error
    assert process.listener is None
    assert process.action_loop_task is None
    assert process.event_send_loop_task is None
    assert process.blocked_main_loop is None
    assert process._stop_event_task is None
    cast(MagicMock, process.worker_id_queue).put.assert_not_called()
    process.finalize_listener_cleanup()


def test_registration_failure_closes_health_server() -> None:
    process = MagicMock()
    process.start_health_server = AsyncMock()
    error = _registration_error()
    process.start = AsyncMock(side_effect=error)
    process.stop_health_server = AsyncMock()

    with patch(f"{LISTENER_MODULE}.WorkerActionListenerProcess", return_value=process):
        with pytest.raises(grpc.aio.AioRpcError) as caught:
            worker_action_listener_process(
                name="test-worker",
                actions=["test-action"],
                slot_config={"default": 1},
                config=MagicMock(),
                action_queue=MagicMock(),
                event_queue=MagicMock(),
                handle_kill=False,
                debug=False,
                labels=[],
                worker_id_queue=MagicMock(),
                stop_event=MagicMock(),
            )

    assert caught.value is error
    process.start_health_server.assert_awaited_once()
    process.stop_health_server.assert_awaited_once()


def _failed_registration_subprocess() -> None:
    config = MagicMock()
    config.healthcheck.enabled = False
    dispatcher = MagicMock()
    dispatcher.get_action_listener = AsyncMock(side_effect=_registration_error())
    with (
        patch(f"{LISTENER_MODULE}.Client"),
        patch(f"{LISTENER_MODULE}.DispatcherClient", return_value=dispatcher),
    ):
        worker_action_listener_process(
            name="test-worker",
            actions=["test-action"],
            slot_config={"default": 1},
            config=config,
            action_queue=MagicMock(),
            event_queue=MagicMock(),
            handle_kill=False,
            debug=False,
            labels=[],
            worker_id_queue=MagicMock(),
            stop_event=MagicMock(),
        )


def test_registration_failure_exits_subprocess_with_original_error(
    capfd: pytest.CaptureFixture[str],
) -> None:
    process = multiprocessing.get_context("spawn").Process(
        target=_failed_registration_subprocess
    )
    process.start()
    try:
        process.join(timeout=10)
        assert process.exitcode == 1
        output = capfd.readouterr()
        assert "registration failed" in output.err
        assert "AttributeError" not in output.err
    finally:
        if process.is_alive():
            process.kill()
            process.join(timeout=5)
        process.close()


@pytest.mark.parametrize("handle_kill", [False, True])
@pytest.mark.parametrize("listener_exitcode", [0, 1, -9])
def test_worker_start_propagates_listener_death_after_cleanup(
    handle_kill: bool, listener_exitcode: int
) -> None:
    worker = _make_worker(handle_kill)
    listener = MagicMock()
    listener.is_alive.return_value = False
    listener.exitcode = listener_exitcode
    worker._action_listener_process = listener

    try:
        with patch.object(worker, "_aio_start", new=lambda: _monitor(worker)):
            with pytest.raises(
                RuntimeError, match="action listener process exited unexpectedly"
            ):
                worker.start()
        assert worker.status == WorkerStatus.UNHEALTHY
        cast(MagicMock, worker._action_runner).exit_gracefully.assert_awaited_once()
        cast(MagicMock, worker._action_runner).cleanup.assert_called_once()
        cast(AsyncMock, worker._cleanup_lifespan).assert_awaited_once()
    finally:
        _cleanup_worker(worker)


def test_task_limit_shutdown_exits_successfully() -> None:
    worker = _make_worker(handle_kill=True)
    worker._config.terminate_worker_after_num_tasks = 1
    listener = MagicMock()
    listener.is_alive.return_value = True
    worker._action_listener_process = listener
    try:
        with (
            patch.object(worker, "_aio_start", new=lambda: _monitor(worker)),
            patch(f"{WORKER_MODULE}.task_count", value=1),
        ):
            with pytest.raises(SystemExit) as caught:
                worker.start()
        assert caught.value.code == 0
        assert worker._aio_start_exception is None
        cast(MagicMock, worker._action_runner).exit_gracefully.assert_awaited_once()
    finally:
        _cleanup_worker(worker)


def _make_worker(handle_kill: bool) -> Worker:
    config = MagicMock()
    config.apply_namespace.return_value = "test-worker"
    config.healthcheck.enabled = False
    config.terminate_worker_after_num_tasks = None
    with (
        patch(f"{WORKER_MODULE}.Client"),
        patch.object(Worker, "_setup_signal_handlers"),
    ):
        worker = Worker("test-worker", config, {"default": 1}, handle_kill=handle_kill)
    worker._pause_task_assignment = AsyncMock()  # type: ignore[method-assign]
    worker._cleanup_lifespan = AsyncMock()  # type: ignore[method-assign]
    runner = MagicMock()
    runner.exit_gracefully = AsyncMock()
    worker._action_runner = runner
    return worker


async def _monitor(worker: Worker) -> None:
    worker._action_listener_health_check = asyncio.create_task(
        worker._check_listener_health()
    )
    await worker._action_listener_health_check


def _cleanup_worker(worker: Worker) -> None:
    worker._close_queues()
    worker._worker_id_queue.close()
    if worker._loop is not None:
        worker._loop.close()
