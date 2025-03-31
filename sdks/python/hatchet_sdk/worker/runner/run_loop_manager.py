import asyncio
import logging
from multiprocessing import Queue
from typing import Any, Literal, TypeVar

from hatchet_sdk.client import Client
from hatchet_sdk.clients.dispatcher.action_listener import Action
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.utils.typing import WorkflowValidator
from hatchet_sdk.worker.action_listener_process import ActionEvent
from hatchet_sdk.worker.runner.runner import Runner
from hatchet_sdk.worker.runner.utils.capture_logs import capture_logs

STOP_LOOP_TYPE = Literal["STOP_LOOP"]
STOP_LOOP: STOP_LOOP_TYPE = "STOP_LOOP"

T = TypeVar("T")


class WorkerActionRunLoopManager:
    def __init__(
        self,
        name: str,
        action_registry: dict[str, Task[Any, Any]],
        validator_registry: dict[str, WorkflowValidator],
        slots: int | None,
        config: ClientConfig,
        action_queue: "Queue[Action | STOP_LOOP_TYPE]",
        event_queue: "Queue[ActionEvent]",
        loop: asyncio.AbstractEventLoop,
        handle_kill: bool = True,
        debug: bool = False,
        labels: dict[str, str | int] = {},
    ) -> None:
        self.name = name
        self.action_registry = action_registry
        self.validator_registry = validator_registry
        self.slots = slots
        self.config = config
        self.action_queue = action_queue
        self.event_queue = event_queue
        self.loop = loop
        self.handle_kill = handle_kill
        self.debug = debug
        self.labels = labels

        if self.debug:
            logger.setLevel(logging.DEBUG)

        self.killing = False
        self.runner: Runner | None = None

        self.client = Client(config=self.config, debug=self.debug)
        self.start()

    def start(self, retry_count: int = 1) -> None:
        k = self.loop.create_task(self.aio_start(retry_count))  # noqa: F841

    async def aio_start(self, retry_count: int = 1) -> None:
        await capture_logs(
            self.client.log_interceptor,
            self.client.event,
            self._async_start,
        )(retry_count=retry_count)

    async def _async_start(self, retry_count: int = 1) -> None:
        logger.info("starting runner...")
        self.loop = asyncio.get_running_loop()
        # needed for graceful termination
        k = self.loop.create_task(self._start_action_loop())
        await k

    def cleanup(self) -> None:
        self.killing = True

        self.action_queue.put(STOP_LOOP)

    async def wait_for_tasks(self) -> None:
        if self.runner:
            await self.runner.wait_for_tasks()

    async def _start_action_loop(self) -> None:
        self.runner = Runner(
            self.event_queue,
            self.config,
            self.slots,
            self.handle_kill,
            self.action_registry,
            self.validator_registry,
            self.labels,
        )

        logger.debug(f"'{self.name}' waiting for {list(self.action_registry.keys())}")
        while not self.killing:
            action = await self._get_action()
            if action == STOP_LOOP:
                logger.debug("stopping action runner loop...")
                break

            self.runner.run(action)
        logger.debug("action runner loop stopped")

    async def _get_action(self) -> Action | STOP_LOOP_TYPE:
        return await self.loop.run_in_executor(None, self.action_queue.get)

    async def exit_gracefully(self) -> None:
        if self.killing:
            return

        logger.info("gracefully exiting runner...")

        self.cleanup()

        # Wait for 1 second to allow last calls to flush. These are calls which have been
        # added to the event loop as callbacks to tasks, so we're not aware of them in the
        # task list.
        await asyncio.sleep(1)

    def exit_forcefully(self) -> None:
        logger.info("forcefully exiting runner...")
        self.cleanup()
