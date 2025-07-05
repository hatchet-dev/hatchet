import asyncio
import multiprocessing
import multiprocessing.context
import os
import re
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

from aiohttp import web
from aiohttp.web_request import Request
from aiohttp.web_response import Response
from prometheus_client import Gauge, generate_latest
from pydantic import BaseModel

from hatchet_sdk.client import Client
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.contracts.v1.workflows_pb2 import CreateWorkflowVersionRequest
from hatchet_sdk.exceptions import LoopAlreadyRunningError
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


class HealthCheckResponse(BaseModel):
    status: str
    name: str
    slots: int
    actions: list[str]
    labels: dict[str, str | int]
    python_version: str


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

        self.loop: asyncio.AbstractEventLoop | None

        self.client = Client(config=self.config, debug=self.debug)

        self._setup_signal_handlers()

        self.worker_status_gauge = Gauge(
            "hatchet_worker_status_" + re.sub(r"\W+", "", name),
            "Current status of the Hatchet worker",
        )

        self.has_any_durable = False
        self.has_any_non_durable = False

        self.lifespan = lifespan
        self.lifespan_stack: AsyncExitStack | None = None

        self.register_workflows(workflows or [])

    def register_workflow_from_opts(self, opts: CreateWorkflowVersionRequest) -> None:
        try:
            self.client.admin.put_workflow(opts)
        except Exception as e:
            logger.error(f"failed to register workflow: {opts.name}")
            logger.error(e)
            sys.exit(1)

    def register_workflow(self, workflow: BaseWorkflow[Any]) -> None:
        if not workflow.tasks:
            raise ValueError(
                "workflow must have at least one task registered before registering"
            )

        try:
            self.client.admin.put_workflow(workflow.to_proto())
        except Exception as e:
            logger.error(f"failed to register workflow: {workflow.name}")
            logger.error(e)
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

        logger.debug("Creating new event loop")
        self.loop = asyncio.new_event_loop()
        asyncio.set_event_loop(self.loop)

    async def _health_check_handler(self, request: Request) -> Response:
        response = HealthCheckResponse(
            status=self.status.name,
            name=self.name,
            slots=self.slots or 0,
            actions=list(self.action_registry.keys()),
            labels=self.labels,
            python_version=sys.version,
        ).model_dump()

        return web.json_response(response)

    async def _metrics_handler(self, request: Request) -> Response:
        self.worker_status_gauge.set(1 if self.status == WorkerStatus.HEALTHY else 0)

        return web.Response(body=generate_latest(), content_type="text/plain")

    async def _start_health_server(self) -> None:
        port = self.config.healthcheck.port

        app = web.Application()
        app.add_routes(
            [
                web.get("/health", self._health_check_handler),
                web.get("/metrics", self._metrics_handler),
            ]
        )

        runner = web.AppRunner(app)

        try:
            await runner.setup()
            await web.TCPSite(runner, "0.0.0.0", port).start()
        except Exception as e:
            logger.error("failed to start healthcheck server")
            logger.error(str(e))
            return

        logger.info(f"healthcheck server running on port {port}")

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

        if self.config.healthcheck.enabled:
            await self._start_health_server()

        lifespan_context = None
        if self.lifespan:
            lifespan_context = await self._setup_lifespan()

        if self.has_any_non_durable:
            self.action_listener_process = self._start_action_listener(is_durable=False)
            self.action_runner = self._run_action_runner(
                is_durable=False, lifespan_context=lifespan_context
            )

        if self.has_any_durable:
            self.durable_action_listener_process = self._start_action_listener(
                is_durable=True
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

        lifespan_gen = self.lifespan()
        try:
            context = await anext(lifespan_gen)
            await self.lifespan_stack.enter_async_context(
                _create_async_context_manager(lifespan_gen)
            )
            return context
        except StopAsyncIteration:
            return None

    async def _cleanup_lifespan(self) -> None:
        if self.lifespan_stack is not None:
            await self.lifespan_stack.aclose()

    def _start_action_listener(
        self, is_durable: bool
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
        except Exception as e:
            logger.error(f"failed to start action listener: {e}")
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
        except Exception as e:
            logger.error(f"error checking listener health: {e}")

    def _setup_signal_handlers(self) -> None:
        signal.signal(signal.SIGTERM, self._handle_exit_signal)
        signal.signal(signal.SIGINT, self._handle_exit_signal)
        signal.signal(signal.SIGQUIT, self._handle_force_quit_signal)

    def _handle_exit_signal(self, signum: int, frame: FrameType | None) -> None:
        sig_name = "SIGTERM" if signum == signal.SIGTERM else "SIGINT"
        logger.info(f"received signal {sig_name}...")
        if self.loop:
            self.loop.create_task(self.exit_gracefully())

    def _handle_force_quit_signal(self, signum: int, frame: FrameType | None) -> None:
        logger.info("received SIGQUIT...")
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

        await self._cleanup_lifespan()

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
