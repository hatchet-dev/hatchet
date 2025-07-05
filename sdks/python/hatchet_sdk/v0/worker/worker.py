import asyncio
import multiprocessing
import multiprocessing.context
import os
import signal
import sys
from concurrent.futures import Future
from dataclasses import dataclass, field
from enum import Enum
from multiprocessing import Queue
from multiprocessing.process import BaseProcess
from types import FrameType
from typing import Any, Callable, TypeVar, get_type_hints

from aiohttp import web
from aiohttp.web_request import Request
from aiohttp.web_response import Response
from prometheus_client import CONTENT_TYPE_LATEST, Gauge, generate_latest

from hatchet_sdk.contracts.workflows_pb2 import CreateWorkflowVersionOpts
from hatchet_sdk.logger import logger
from hatchet_sdk.v0 import Context
from hatchet_sdk.v0.client import Client, new_client_raw
from hatchet_sdk.v0.loader import ClientConfig
from hatchet_sdk.v0.utils.types import WorkflowValidator
from hatchet_sdk.v0.utils.typing import is_basemodel_subclass
from hatchet_sdk.v0.v2.callable import HatchetCallable
from hatchet_sdk.v0.v2.concurrency import ConcurrencyFunction
from hatchet_sdk.v0.worker.action_listener_process import worker_action_listener_process
from hatchet_sdk.v0.worker.runner.run_loop_manager import WorkerActionRunLoopManager
from hatchet_sdk.v0.workflow import WorkflowInterface

T = TypeVar("T")


class WorkerStatus(Enum):
    INITIALIZED = 1
    STARTING = 2
    HEALTHY = 3
    UNHEALTHY = 4


@dataclass
class WorkerStartOptions:
    loop: asyncio.AbstractEventLoop | None = field(default=None)


TWorkflow = TypeVar("TWorkflow", bound=object)


class Worker:
    def __init__(
        self,
        name: str,
        config: ClientConfig = ClientConfig(),
        max_runs: int | None = None,
        labels: dict[str, str | int] = {},
        debug: bool = False,
        owned_loop: bool = True,
        handle_kill: bool = True,
    ) -> None:
        self.name = name
        self.config = config
        self.max_runs = max_runs
        self.debug = debug
        self.labels = labels
        self.handle_kill = handle_kill
        self.owned_loop = owned_loop

        self.client: Client

        self.action_registry: dict[str, Callable[[Context], Any]] = {}
        self.validator_registry: dict[str, WorkflowValidator] = {}

        self.killing: bool = False
        self._status: WorkerStatus

        self.action_listener_process: BaseProcess
        self.action_listener_health_check: asyncio.Task[Any]
        self.action_runner: WorkerActionRunLoopManager

        self.ctx = multiprocessing.get_context("spawn")

        self.action_queue: "Queue[Any]" = self.ctx.Queue()
        self.event_queue: "Queue[Any]" = self.ctx.Queue()

        self.loop: asyncio.AbstractEventLoop

        self.client = new_client_raw(self.config, self.debug)
        self.name = self.client.config.namespace + self.name

        self._setup_signal_handlers()

        self.worker_status_gauge = Gauge(
            "hatchet_worker_status", "Current status of the Hatchet worker"
        )

    def register_function(self, action: str, func: Callable[[Context], Any]) -> None:
        self.action_registry[action] = func

    def register_workflow_from_opts(
        self, name: str, opts: CreateWorkflowVersionOpts
    ) -> None:
        try:
            self.client.admin.put_workflow(opts.name, opts)
        except Exception as e:
            logger.error(f"failed to register workflow: {opts.name}")
            logger.error(e)
            sys.exit(1)

    def register_workflow(self, workflow: TWorkflow) -> None:
        ## Hack for typing
        assert isinstance(workflow, WorkflowInterface)

        namespace = self.client.config.namespace

        try:
            self.client.admin.put_workflow(
                workflow.get_name(namespace), workflow.get_create_opts(namespace)
            )
        except Exception as e:
            logger.error(f"failed to register workflow: {workflow.get_name(namespace)}")
            logger.error(e)
            sys.exit(1)

        def create_action_function(
            action_func: Callable[..., T]
        ) -> Callable[[Context], T]:
            def action_function(context: Context) -> T:
                return action_func(workflow, context)

            if asyncio.iscoroutinefunction(action_func):
                setattr(action_function, "is_coroutine", True)
            else:
                setattr(action_function, "is_coroutine", False)

            return action_function

        for action_name, action_func in workflow.get_actions(namespace):
            self.action_registry[action_name] = create_action_function(action_func)
            return_type = get_type_hints(action_func).get("return")

            self.validator_registry[action_name] = WorkflowValidator(
                workflow_input=workflow.input_validator,
                step_output=return_type if is_basemodel_subclass(return_type) else None,
            )

    def status(self) -> WorkerStatus:
        return self._status

    def setup_loop(self, loop: asyncio.AbstractEventLoop | None = None) -> bool:
        try:
            loop = loop or asyncio.get_running_loop()
            self.loop = loop
            created_loop = False
            logger.debug("using existing event loop")
            return created_loop
        except RuntimeError:
            self.loop = asyncio.new_event_loop()
            logger.debug("creating new event loop")
            asyncio.set_event_loop(self.loop)
            created_loop = True
            return created_loop

    async def health_check_handler(self, request: Request) -> Response:
        status = self.status()

        return web.json_response({"status": status.name})

    async def metrics_handler(self, request: Request) -> Response:
        self.worker_status_gauge.set(1 if self.status() == WorkerStatus.HEALTHY else 0)

        return web.Response(body=generate_latest(), content_type="text/plain")

    async def start_health_server(self) -> None:
        port = self.config.worker_healthcheck_port or 8001

        app = web.Application()
        app.add_routes(
            [
                web.get("/health", self.health_check_handler),
                web.get("/metrics", self.metrics_handler),
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

    def start(
        self, options: WorkerStartOptions = WorkerStartOptions()
    ) -> Future[asyncio.Task[None]]:
        self.owned_loop = self.setup_loop(options.loop)

        f = asyncio.run_coroutine_threadsafe(
            self._async_start(options, _from_start=True), self.loop
        )

        # start the loop and wait until its closed
        if self.owned_loop:
            self.loop.run_forever()

            if self.handle_kill:
                sys.exit(0)

        return f

    ## Start methods
    async def _async_start(
        self,
        options: WorkerStartOptions = WorkerStartOptions(),
        _from_start: bool = False,
    ) -> asyncio.Task[None]:
        main_pid = os.getpid()
        logger.info("------------------------------------------")
        logger.info("STARTING HATCHET...")
        logger.debug(f"worker runtime starting on PID: {main_pid}")

        self._status = WorkerStatus.STARTING

        if len(self.action_registry.keys()) == 0:
            raise ValueError(
                "no actions registered, register workflows or actions before starting worker"
            )

        # non blocking setup
        if not _from_start:
            self.setup_loop(options.loop)

        if self.config.worker_healthcheck_enabled:
            await self.start_health_server()

        self.action_listener_process = self._start_listener()

        self.action_runner = self._run_action_runner()

        self.action_listener_health_check = self.loop.create_task(
            self._check_listener_health()
        )

        return await self.action_listener_health_check

    def _run_action_runner(self) -> WorkerActionRunLoopManager:
        # Retrieve the shared queue
        return WorkerActionRunLoopManager(
            self.name,
            self.action_registry,
            self.validator_registry,
            self.max_runs,
            self.config,
            self.action_queue,
            self.event_queue,
            self.loop,
            self.handle_kill,
            self.client.debug,
            self.labels,
        )

    def _start_listener(self) -> multiprocessing.context.SpawnProcess:
        action_list = [str(key) for key in self.action_registry.keys()]

        try:
            process = self.ctx.Process(
                target=worker_action_listener_process,
                args=(
                    self.name,
                    action_list,
                    self.max_runs,
                    self.config,
                    self.action_queue,
                    self.event_queue,
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
                    self.action_listener_process is None
                    or not self.action_listener_process.is_alive()
                ):
                    logger.debug("child action listener process killed...")
                    self._status = WorkerStatus.UNHEALTHY
                    if not self.killing:
                        self.loop.create_task(self.exit_gracefully())
                    break
                else:
                    self._status = WorkerStatus.HEALTHY
                await asyncio.sleep(1)
        except Exception as e:
            logger.error(f"error checking listener health: {e}")

    ## Cleanup methods
    def _setup_signal_handlers(self) -> None:
        signal.signal(signal.SIGTERM, self._handle_exit_signal)
        signal.signal(signal.SIGINT, self._handle_exit_signal)
        signal.signal(signal.SIGQUIT, self._handle_force_quit_signal)

    def _handle_exit_signal(self, signum: int, frame: FrameType | None) -> None:
        sig_name = "SIGTERM" if signum == signal.SIGTERM else "SIGINT"
        logger.info(f"received signal {sig_name}...")
        self.loop.create_task(self.exit_gracefully())

    def _handle_force_quit_signal(self, signum: int, frame: FrameType | None) -> None:
        logger.info("received SIGQUIT...")
        self.exit_forcefully()

    async def close(self) -> None:
        logger.info(f"closing worker '{self.name}'...")
        self.killing = True
        # self.action_queue.close()
        # self.event_queue.close()

        if self.action_runner is not None:
            self.action_runner.cleanup()

        await self.action_listener_health_check

    async def exit_gracefully(self) -> None:
        logger.debug(f"gracefully stopping worker: {self.name}")

        if self.killing:
            return self.exit_forcefully()

        self.killing = True

        await self.action_runner.wait_for_tasks()

        await self.action_runner.exit_gracefully()

        if self.action_listener_process and self.action_listener_process.is_alive():
            self.action_listener_process.kill()

        await self.close()
        if self.loop and self.owned_loop:
            self.loop.stop()

        logger.info("👋")

    def exit_forcefully(self) -> None:
        self.killing = True

        logger.debug(f"forcefully stopping worker: {self.name}")

        self.close()

        if self.action_listener_process:
            self.action_listener_process.kill()  # Forcefully kill the process

        logger.info("👋")
        sys.exit(
            1
        )  # Exit immediately TODO - should we exit with 1 here, there may be other workers to cleanup


def register_on_worker(callable: HatchetCallable[T], worker: Worker) -> None:
    worker.register_function(callable.get_action_name(), callable)

    if callable.function_on_failure is not None:
        worker.register_function(
            callable.function_on_failure.get_action_name(), callable.function_on_failure
        )

    if callable.function_concurrency is not None:
        worker.register_function(
            callable.function_concurrency.get_action_name(),
            callable.function_concurrency,
        )

    opts = callable.to_workflow_opts()

    worker.register_workflow_from_opts(opts.name, opts)
