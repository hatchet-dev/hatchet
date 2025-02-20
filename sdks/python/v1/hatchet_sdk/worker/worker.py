import asyncio
import multiprocessing
import multiprocessing.context
import os
import signal
import sys
from dataclasses import dataclass, field
from enum import Enum
from multiprocessing import Queue
from multiprocessing.process import BaseProcess
from types import FrameType
from typing import TYPE_CHECKING, Any, TypeVar, Union, get_type_hints

from aiohttp import web
from aiohttp.web_request import Request
from aiohttp.web_response import Response
from prometheus_client import Gauge, generate_latest

from hatchet_sdk.client import Client, new_client_raw
from hatchet_sdk.clients.dispatcher.action_listener import Action
from hatchet_sdk.contracts.workflows_pb2 import CreateWorkflowVersionOpts
from hatchet_sdk.loader import ClientConfig
from hatchet_sdk.logger import logger
from hatchet_sdk.utils.types import WorkflowValidator
from hatchet_sdk.utils.typing import is_basemodel_subclass
from hatchet_sdk.worker.action_listener_process import (
    ActionEvent,
    worker_action_listener_process,
)
from hatchet_sdk.worker.runner.run_loop_manager import (
    STOP_LOOP_TYPE,
    WorkerActionRunLoopManager,
)
from hatchet_sdk.workflow import Step

if TYPE_CHECKING:
    from hatchet_sdk.workflow import BaseWorkflow

T = TypeVar("T")


class WorkerStatus(Enum):
    INITIALIZED = 1
    STARTING = 2
    HEALTHY = 3
    UNHEALTHY = 4


@dataclass
class WorkerStartOptions:
    loop: asyncio.AbstractEventLoop | None = field(default=None)


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

        self.action_registry: dict[str, Step[Any]] = {}
        self.validator_registry: dict[str, WorkflowValidator] = {}

        self.killing: bool = False
        self._status: WorkerStatus

        self.action_listener_process: BaseProcess
        self.action_listener_health_check: asyncio.Task[None]
        self.action_runner: WorkerActionRunLoopManager

        self.ctx = multiprocessing.get_context("spawn")

        self.action_queue: "Queue[Action | STOP_LOOP_TYPE]" = self.ctx.Queue()
        self.event_queue: "Queue[ActionEvent]" = self.ctx.Queue()

        self.loop: asyncio.AbstractEventLoop

        self.client = new_client_raw(self.config, self.debug)
        self.name = self.client.config.namespace + self.name

        self._setup_signal_handlers()

        self.worker_status_gauge = Gauge(
            "hatchet_worker_status", "Current status of the Hatchet worker"
        )

    def register_workflow_from_opts(
        self, name: str, opts: CreateWorkflowVersionOpts
    ) -> None:
        try:
            self.client.admin.put_workflow(opts.name, opts)
        except Exception as e:
            logger.error(f"failed to register workflow: {opts.name}")
            logger.error(e)
            sys.exit(1)

    def register_workflow(self, workflow: Union["BaseWorkflow", Any]) -> None:
        namespace = self.client.config.namespace

        try:
            self.client.admin.put_workflow(
                workflow.get_name(namespace), workflow.get_create_opts(namespace)
            )
        except Exception as e:
            logger.error(f"failed to register workflow: {workflow.get_name(namespace)}")
            logger.error(e)
            sys.exit(1)

        for step in workflow.steps:
            action_name = workflow.create_action_name(namespace, step)
            self.action_registry[action_name] = step
            return_type = get_type_hints(step.fn).get("return")

            self.validator_registry[action_name] = WorkflowValidator(
                workflow_input=workflow.config.input_validator,
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
        port = self.config.healthcheck.port

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

    def start(self, options: WorkerStartOptions = WorkerStartOptions()) -> None:
        self.owned_loop = self.setup_loop(options.loop)

        asyncio.run_coroutine_threadsafe(
            self.aio_start(options, _from_start=True), self.loop
        )

        # start the loop and wait until its closed
        if self.owned_loop:
            self.loop.run_forever()

            if self.handle_kill:
                sys.exit(0)

    ## Start methods
    async def aio_start(
        self,
        options: WorkerStartOptions = WorkerStartOptions(),
        _from_start: bool = False,
    ) -> None:
        main_pid = os.getpid()
        logger.info("------------------------------------------")
        logger.info("STARTING HATCHET...")
        logger.debug(f"worker runtime starting on PID: {main_pid}")

        self._status = WorkerStatus.STARTING

        if len(self.action_registry.keys()) == 0:
            logger.error(
                "no actions registered, register workflows or actions before starting worker"
            )
            return None

        # non blocking setup
        if not _from_start:
            self.setup_loop(options.loop)

        if self.config.healthcheck.enabled:
            await self.start_health_server()

        self.action_listener_process = self._start_listener()

        self.action_runner = self._run_action_runner()

        self.action_listener_health_check = self.loop.create_task(
            self._check_listener_health()
        )

        await self.action_listener_health_check

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

        ## TODO: `self.close` needs to be awaited / used
        self.close()  # type: ignore[unused-coroutine]

        if self.action_listener_process:
            self.action_listener_process.kill()  # Forcefully kill the process

        logger.info("👋")
        sys.exit(
            1
        )  # Exit immediately TODO - should we exit with 1 here, there may be other workers to cleanup
