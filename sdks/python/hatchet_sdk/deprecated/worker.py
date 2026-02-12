"""Legacy dual-worker orchestration for pre-slot-config engines.

When connected to an older Hatchet engine that does not support multiple slot types,
this module provides the old worker start flow which:
  - Splits tasks into durable and non-durable registries
  - Spawns separate action listener processes for each
  - Creates separate action runners for each
  - Monitors health of both processes
"""

from __future__ import annotations

import asyncio
import multiprocessing.context
import os
import sys
from multiprocessing import Queue
from typing import TYPE_CHECKING, Any

from hatchet_sdk.deprecated.action_listener_process import (
    legacy_worker_action_listener_process,
)
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import Action
from hatchet_sdk.runnables.contextvars import task_count
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.utils.typing import STOP_LOOP_TYPE
from hatchet_sdk.worker.action_listener_process import ActionEvent
from hatchet_sdk.worker.runner.run_loop_manager import WorkerActionRunLoopManager

if TYPE_CHECKING:
    from hatchet_sdk.worker.worker import Worker


async def legacy_aio_start(worker: Worker) -> None:
    """Start the worker using the legacy dual-worker architecture.

    This is the old _aio_start flow that splits durable and non-durable tasks
    into separate processes, for engines that don't understand slot_config.
    """
    from hatchet_sdk.exceptions import LifespanSetupError
    from hatchet_sdk.worker.worker import WorkerStatus

    main_pid = os.getpid()

    logger.info("------------------------------------------")
    logger.info("STARTING HATCHET (legacy mode)...")
    logger.debug(f"worker runtime starting on PID: {main_pid}")

    worker._status = WorkerStatus.STARTING

    if len(worker.action_registry.keys()) == 0:
        raise ValueError(
            "no actions registered, register workflows or actions before starting worker"
        )

    # Split the unified action registry into durable/non-durable
    durable_action_registry: dict[str, Task[Any, Any]] = {}
    non_durable_action_registry: dict[str, Task[Any, Any]] = {}

    for action_name, task in worker.action_registry.items():
        if task.is_durable:
            durable_action_registry[action_name] = task
        else:
            non_durable_action_registry[action_name] = task

    has_any_non_durable = len(non_durable_action_registry) > 0
    has_any_durable = len(durable_action_registry) > 0

    # Create separate queues for durable workers
    durable_action_queue: Queue[Action | STOP_LOOP_TYPE] = worker.ctx.Queue()
    durable_event_queue: Queue[ActionEvent] = worker.ctx.Queue()

    lifespan_context = None
    if worker.lifespan:
        try:
            lifespan_context = await worker._setup_lifespan()
        except LifespanSetupError as e:
            logger.exception("lifespan setup failed")
            if worker.loop:
                worker.loop.stop()
            raise e

    # Slot conversion: use default and durable from slot_config
    default_slots = worker.slot_config.get("default", 100)
    durable_slots = worker.slot_config.get("durable", 1000)

    durable_action_listener_process = None
    durable_action_runner = None

    if has_any_non_durable:
        worker.action_listener_process = _legacy_start_action_listener(
            worker,
            is_durable=False,
            actions=list(non_durable_action_registry.keys()),
            slots=default_slots,
            action_queue=worker.action_queue,
            event_queue=worker.event_queue,
        )
        worker.action_runner = _legacy_run_action_runner(
            worker,
            name_suffix="",
            action_registry=non_durable_action_registry,
            max_runs=default_slots,
            action_queue=worker.action_queue,
            event_queue=worker.event_queue,
            lifespan_context=lifespan_context,
        )

    if has_any_durable:
        durable_action_listener_process = _legacy_start_action_listener(
            worker,
            is_durable=True,
            actions=list(durable_action_registry.keys()),
            slots=durable_slots,
            action_queue=durable_action_queue,
            event_queue=durable_event_queue,
        )
        durable_action_runner = _legacy_run_action_runner(
            worker,
            name_suffix="_durable",
            action_registry=durable_action_registry,
            max_runs=durable_slots,
            action_queue=durable_action_queue,
            event_queue=durable_event_queue,
            lifespan_context=lifespan_context,
        )

    if worker.loop:
        # Store references for cleanup BEFORE the health check blocks,
        # so they are available when exit_gracefully() runs.
        worker.durable_action_listener_process = durable_action_listener_process
        worker.durable_action_queue = durable_action_queue
        worker.durable_event_queue = durable_event_queue
        worker._legacy_durable_action_runner = durable_action_runner  # type: ignore[attr-defined]

        worker._lifespan_cleanup_complete = asyncio.Event()
        worker.action_listener_health_check = worker.loop.create_task(
            _legacy_check_listener_health(
                worker,
                durable_action_listener_process,
            )
        )

        await worker.action_listener_health_check

        try:
            await worker._cleanup_lifespan()
        except Exception:
            logger.exception("lifespan cleanup failed")
        finally:
            worker._lifespan_cleanup_complete.set()


def _legacy_start_action_listener(
    worker: Worker,
    is_durable: bool,
    actions: list[str],
    slots: int,
    action_queue: Queue[Any],
    event_queue: Queue[Any],
) -> multiprocessing.context.SpawnProcess:
    try:
        process = worker.ctx.Process(
            target=legacy_worker_action_listener_process,
            args=(
                worker.name + ("_durable" if is_durable else ""),
                actions,
                slots,
                worker.config,
                action_queue,
                event_queue,
                worker.handle_kill,
                worker.client.debug,
                worker.labels,
            ),
        )
        process.start()
        logger.debug(
            f"legacy action listener ({'durable' if is_durable else 'non-durable'}) starting on PID: {process.pid}"
        )
        return process
    except Exception:
        logger.exception("failed to start legacy action listener")
        sys.exit(1)


def _legacy_run_action_runner(
    worker: Worker,
    name_suffix: str,
    action_registry: dict[str, Task[Any, Any]],
    max_runs: int,
    action_queue: Queue[Any],
    event_queue: Queue[Any],
    lifespan_context: Any | None,
) -> WorkerActionRunLoopManager:
    if worker.loop:
        return WorkerActionRunLoopManager(
            worker.name + name_suffix,
            action_registry,
            max_runs,
            worker.config,
            action_queue,
            event_queue,
            worker.loop,
            worker.handle_kill,
            worker.client.debug,
            worker.labels,
            lifespan_context,
        )

    raise RuntimeError("event loop not set, cannot start action runner")


async def _legacy_check_listener_health(
    worker: Worker,
    durable_action_listener_process: multiprocessing.context.SpawnProcess | None,
) -> None:
    from hatchet_sdk.worker.worker import WorkerStatus

    logger.debug("starting legacy action listener health check...")
    try:
        while not worker.killing:
            if (
                not worker.action_listener_process
                and not durable_action_listener_process
            ) or (
                worker.action_listener_process
                and durable_action_listener_process
                and not worker.action_listener_process.is_alive()
                and not durable_action_listener_process.is_alive()
            ):
                logger.debug("child action listener process killed...")
                worker._status = WorkerStatus.UNHEALTHY
                if worker.loop:
                    worker.loop.create_task(worker.exit_gracefully())
                break

            if (
                worker.config.terminate_worker_after_num_tasks
                and task_count.value >= worker.config.terminate_worker_after_num_tasks
            ):
                if worker.loop:
                    worker.loop.create_task(worker.exit_gracefully())
                break

            worker._status = WorkerStatus.HEALTHY
            await asyncio.sleep(1)
    except Exception:
        logger.exception("error checking listener health")
