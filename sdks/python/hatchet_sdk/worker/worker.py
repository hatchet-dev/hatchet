import asyncio
import multiprocessing
import multiprocessing.context
import os
import signal
import sys
from collections.abc import AsyncGenerator, Callable
from contextlib import AsyncExitStack, asynccontextmanager, suppress
from dataclasses import dataclass, field
from enum import Enum
from multiprocessing import Queue
from multiprocessing.process import BaseProcess
from types import FrameType
from typing import Any, TypeVar
from warnings import warn

from hatchet_sdk.client import Client
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.contracts.v1.workflows_pb2 import CreateWorkflowVersionRequest
from hatchet_sdk.exceptions import LifespanSetupError, LoopAlreadyRunningError
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import Action
from hatchet_sdk.runnables.contextvars import task_count
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.runnables.workflow import BaseWorkflow
from hatchet_sdk.utils.typing import STOP_LOOP_TYPE
from hatchet_sdk.worker.action_listener_process import (
    ActionEvent,
    worker_action_listener_process,
)
from hatchet_sdk.worker.runner.run_loop_manager import WorkerActionRunLoopManager

T = TypeVar("T")


class WorkerStatus(Enum):
    INITIALIZED = 1
    STARTING = 2
    HEALTHY = 3
    UNHEALTHY = 4


@dataclass
class WorkerStartOptions:
    loop: asyncio.AbstractEventLoop | None = field(default=None)


LifespanGenerator = AsyncGenerator[Any, Any]
LifespanFn = Callable[[], LifespanGenerator]


@asynccontextmanager
async def _create_async_context_manager(
    gen: LifespanGenerator,
) -> AsyncGenerator[None, None]:
    try:
        yield
    finally:
        with suppress(StopAsyncIteration):
            await anext(gen)


class Worker:
    def __init__(
        self,
        name: str,
        config: ClientConfig,
        slots: int,
        durable_slots: int,
        labels: dict[str, str | int] | None = None,
        debug: bool = False,
        owned_loop: bool = True,
        handle_kill: bool = True,
        workflows: list[BaseWorkflow[Any]] | None = None,
        lifespan: LifespanFn | None = None,
    ) -> None:
        self.config = config
        self.name = self.config.apply_namespace(name)
        self.slots = slots
        self.durable_slots = durable_slots
        self.debug = debug
        self.labels = labels or {}
        self.handle_kill = handle_kill
        self.owned_loop = owned_loop

        self.action_registry: dict[str, Task[Any, Any]] = {}
        self.durable_action_registry: dict[str, Task[Any, Any]] = {}

        self.killing: bool = False
        self._status: WorkerStatus = WorkerStatus.INITIALIZED

        self.action_listener_process: BaseProcess | None = None
        self.durable_action_listener_process: BaseProcess | None = None

        self.action_listener_health_check: asyncio.Task[None]

        self.action_runner: WorkerActionRunLoopManager | None = None
        self.durable_action_runner: WorkerActionRunLoopManager | None = None

        self.ctx = multiprocessing.get_context("spawn")

        self.action_queue: Queue[Action | STOP_LOOP_TYPE] = self.ctx.Queue()
        self.event_queue: Queue[ActionEvent] = self.ctx.Queue()

        self.durable_action_queue: Queue[Action | STOP_LOOP_TYPE] = self.ctx.Queue()
        self.durable_event_queue: Queue[ActionEvent] = self.ctx.Queue()

        self.loop: asyncio.AbstractEventLoop | None = None

        self.client = Client(config=self.config, debug=self.debug)

        self._setup_signal_handlers()

        self.has_any_durable = False
        self.has_any_non_durable = False

        self.lifespan = lifespan
        self.lifespan_stack: AsyncExitStack | None = None

        self.register_workflows(workflows or [])

    def register_workflow_from_opts(self, opts: CreateWorkflowVersionRequest) -> None:
        try:
            self.client.admin.put_workflow(opts)
        except Exception:
            logger.exception(f"failed to register workflow: {opts.name}")
            sys.exit(1)

    def register_workflow(self, workflow: BaseWorkflow[Any]) -> None:
        if not workflow.tasks:
            raise ValueError(
                "workflow must have at least one task registered before registering"
            )

        try:
            self.client.admin.put_workflow(workflow.to_proto())
        except Exception:
            logger.exception(f"failed to register workflow: {workflow.name}")
            sys.exit(1)

        for step in workflow.tasks:
            action_name = workflow._create_action_name(step)

            if step.is_durable:
                self.has_any_durable = True
                self.durable_action_registry[action_name] = step
            else:
                self.has_any_non_durable = True
                self.action_registry[action_name] = step

    def register_workflows(self, workflows: list[BaseWorkflow[Any]]) -> None:
        for workflow in workflows:
            self.register_workflow(workflow)

    @property
    def status(self) -> WorkerStatus:
        return self._status

    def _setup_loop(self) -> None:
        try:
            asyncio.get_running_loop()
            raise LoopAlreadyRunningError(
                "An event loop is already running. This worker requires its own dedicated event loop. "
                "Make sure you're not using asyncio.run() or other loop-creating functions in the main thread."
            )
        except RuntimeError:
            pass

        logger.debug("creating new event loop")
        self.loop = asyncio.new_event_loop()
        asyncio.set_event_loop(self.loop)

    def start(self, options: WorkerStartOptions = WorkerStartOptions()) -> None:
        if not (self.action_registry or self.durable_action_registry):
            raise ValueError(
                "no actions registered, register workflows before starting worker"
            )

        if options.loop is not None:
            warn(
                "Passing a custom event loop is deprecated and will be removed in the future. This option no longer has any effect",
                DeprecationWarning,
                stacklevel=1,
            )

        self._setup_loop()

        if not self.loop:
            raise RuntimeError("event loop not set, cannot start worker")

        asyncio.run_coroutine_threadsafe(self._aio_start(), self.loop)

        # start the loop and wait until its closed
        self.loop.run_forever()

        if self.handle_kill:
            sys.exit(0)

    async def _aio_start(self) -> None:
        main_pid = os.getpid()

        logger.info("------------------------------------------")
        logger.info("STARTING HATCHET...")
        logger.debug(f"worker runtime starting on PID: {main_pid}")

        self._status = WorkerStatus.STARTING

        if (
            len(self.action_registry.keys()) == 0
            and len(self.durable_action_registry.keys()) == 0
        ):
            raise ValueError(
                "no actions registered, register workflows or actions before starting worker"
            )

        lifespan_context = None
        if self.lifespan:
            try:
                lifespan_context = await self._setup_lifespan()
            except LifespanSetupError as e:
                logger.exception("lifespan setup failed")
                if self.loop:
                    self.loop.stop()
                raise e

        # Healthcheck server is started inside the spawned action-listener process
        # (non-durable preferred) to avoid being affected by the main worker loop.
        healthcheck_port = self.config.healthcheck.port
        enable_health_server_non_durable = (
            self.config.healthcheck.enabled and self.has_any_non_durable
        )
        enable_health_server_durable = (
            self.config.healthcheck.enabled
            and (not self.has_any_non_durable)
            and self.has_any_durable
        )

        if self.has_any_non_durable:
            self.action_listener_process = self._start_action_listener(
                is_durable=False,
                enable_health_server=enable_health_server_non_durable,
                healthcheck_port=healthcheck_port,
            )
            self.action_runner = self._run_action_runner(
                is_durable=False, lifespan_context=lifespan_context
            )

        if self.has_any_durable:
            self.durable_action_listener_process = self._start_action_listener(
                is_durable=True,
                enable_health_server=enable_health_server_durable,
                healthcheck_port=healthcheck_port,
            )
            self.durable_action_runner = self._run_action_runner(
                is_durable=True, lifespan_context=lifespan_context
            )

        if self.loop:
            self.action_listener_health_check = self.loop.create_task(
                self._check_listener_health()
            )

            await self.action_listener_health_check

    def _run_action_runner(
        self, is_durable: bool, lifespan_context: Any | None
    ) -> WorkerActionRunLoopManager:
        # Retrieve the shared queue
        if self.loop:
            return WorkerActionRunLoopManager(
                self.name + ("_durable" if is_durable else ""),
                self.durable_action_registry if is_durable else self.action_registry,
                self.durable_slots if is_durable else self.slots,
                self.config,
                self.durable_action_queue if is_durable else self.action_queue,
                self.durable_event_queue if is_durable else self.event_queue,
                self.loop,
                self.handle_kill,
                self.client.debug,
                self.labels,
                lifespan_context,
            )

        raise RuntimeError("event loop not set, cannot start action runner")

    async def _setup_lifespan(self) -> Any:
        if self.lifespan is None:
            return None

        self.lifespan_stack = AsyncExitStack()

        try:
            lifespan_gen = self.lifespan()
            context = await anext(lifespan_gen)
            await self.lifespan_stack.enter_async_context(
                _create_async_context_manager(lifespan_gen)
            )
            return context
        except StopAsyncIteration:
            return None
        except Exception as e:
            raise LifespanSetupError("An error occurred during lifespan setup") from e

    async def _cleanup_lifespan(self) -> None:
        try:
            if self.lifespan_stack is not None:
                await self.lifespan_stack.aclose()
        except Exception as e:
            logger.exception("error during lifespan cleanup")
            raise LifespanSetupError("An error occurred during lifespan cleanup") from e

    def _start_action_listener(
        self,
        is_durable: bool,
        *,
        enable_health_server: bool = False,
        healthcheck_port: int = 8001,
    ) -> multiprocessing.context.SpawnProcess:
        try:
            process = self.ctx.Process(
                target=worker_action_listener_process,
                args=(
                    self.name + ("_durable" if is_durable else ""),
                    (
                        list(self.durable_action_registry.keys())
                        if is_durable
                        else list(self.action_registry.keys())
                    ),
                    self.durable_slots if is_durable else self.slots,
                    self.config,
                    self.durable_action_queue if is_durable else self.action_queue,
                    self.durable_event_queue if is_durable else self.event_queue,
                    self.handle_kill,
                    self.client.debug,
                    self.labels,
                ),
            )
            process.start()
            logger.debug(f"action listener starting on PID: {process.pid}")

            return process
        except Exception:
            logger.exception("failed to start action listener")
            sys.exit(1)

    async def _check_listener_health(self) -> None:
        logger.debug("starting action listener health check...")
        try:
            while not self.killing:
                if (
                    not self.action_listener_process
                    and not self.durable_action_listener_process
                ) or (
                    self.action_listener_process
                    and self.durable_action_listener_process
                    and not self.action_listener_process.is_alive()
                    and not self.durable_action_listener_process.is_alive()
                ):
                    logger.debug("child action listener process killed...")
                    self._status = WorkerStatus.UNHEALTHY
                    if self.loop:
                        self.loop.create_task(self.exit_gracefully())
                    break

                if (
                    self.config.terminate_worker_after_num_tasks
                    and task_count.value >= self.config.terminate_worker_after_num_tasks
                ):
                    if self.loop:
                        self.loop.create_task(self.exit_gracefully())
                    break

                self._status = WorkerStatus.HEALTHY
                await asyncio.sleep(1)
        except Exception:
            logger.exception("error checking listener health")

    def _setup_signal_handlers(self) -> None:
        signal.signal(
            signal.SIGTERM,
            (
                self._handle_force_quit_signal
                if self.config.force_shutdown_on_shutdown_signal
                else self._handle_exit_signal
            ),
        )
        signal.signal(
            signal.SIGINT,
            (
                self._handle_force_quit_signal
                if self.config.force_shutdown_on_shutdown_signal
                else self._handle_exit_signal
            ),
        )
        signal.signal(signal.SIGQUIT, self._handle_force_quit_signal)

    def _handle_exit_signal(self, signum: int, frame: FrameType | None) -> None:
        sig_name = "SIGTERM" if signum == signal.SIGTERM else "SIGINT"
        logger.info(f"received signal {sig_name}...")
        if self.loop:
            self.loop.create_task(self.exit_gracefully())

    def _handle_force_quit_signal(self, signum: int, frame: FrameType | None) -> None:
        signal_received = signal.Signals(signum).name
        logger.info(f"received {signal_received}...")
        if self.loop:
            self.loop.create_task(self._exit_forcefully())

    async def _close(self) -> None:
        logger.info(f"closing worker '{self.name}'...")
        self.killing = True

        if self.action_runner is not None:
            self.action_runner.cleanup()

        if self.durable_action_runner is not None:
            self.durable_action_runner.cleanup()

        await self.action_listener_health_check

    async def exit_gracefully(self) -> None:
        logger.debug(f"gracefully stopping worker: {self.name}")

        if self.killing:
            return await self._exit_forcefully()

        self.killing = True

        if self.action_runner:
            await self.action_runner.wait_for_tasks()
            await self.action_runner.exit_gracefully()

        if self.durable_action_runner:
            await self.durable_action_runner.wait_for_tasks()
            await self.durable_action_runner.exit_gracefully()

        if self.action_listener_process and self.action_listener_process.is_alive():
            self.action_listener_process.kill()

        if (
            self.durable_action_listener_process
            and self.durable_action_listener_process.is_alive()
        ):
            self.durable_action_listener_process.kill()

        try:
            await self._cleanup_lifespan()
        except LifespanSetupError:
            logger.exception("lifespan cleanup failed")

        await self._close()
        if self.loop and self.owned_loop:
            self.loop.stop()

        logger.info("👋")

    async def _exit_forcefully(self) -> None:
        self.killing = True

        logger.debug(f"forcefully stopping worker: {self.name}")

        await self._close()

        if self.action_listener_process:
            self.action_listener_process.kill()

        if self.durable_action_listener_process:
            self.durable_action_listener_process.kill()

        logger.info("👋")
        sys.exit(1)
