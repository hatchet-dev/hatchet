import asyncio
import logging
import signal
import time
from dataclasses import dataclass
from multiprocessing import Queue
from typing import Any

import grpc
from grpc.aio import UnaryUnaryCall

from hatchet_sdk.client import Client
from hatchet_sdk.clients.dispatcher.action_listener import (
    ActionListener,
    GetActionListenerRequest,
)
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.rest.models.update_worker_request import UpdateWorkerRequest
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.contracts.dispatcher_pb2 import (
    GROUP_KEY_EVENT_TYPE_STARTED,
    STEP_EVENT_TYPE_STARTED,
    ActionEventResponse,
    GroupKeyActionEvent,
    StepActionEvent,
)
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import Action, ActionType
from hatchet_sdk.runnables.contextvars import (
    ctx_action_key,
    ctx_step_run_id,
    ctx_worker_id,
    ctx_workflow_run_id,
)
from hatchet_sdk.utils.backoff import exp_backoff_sleep
from hatchet_sdk.utils.typing import STOP_LOOP, STOP_LOOP_TYPE

ACTION_EVENT_RETRY_COUNT = 5


@dataclass
class ActionEvent:
    action: Action
    type: Any  # TODO type
    payload: str
    should_not_retry: bool


BLOCKED_THREAD_WARNING = "THE TIME TO START THE TASK RUN IS TOO LONG, THE EVENT LOOP MAY BE BLOCKED. See https://docs.hatchet.run/blog/warning-event-loop-blocked for details and debugging help."


class WorkerActionListenerProcess:
    def __init__(
        self,
        name: str,
        actions: list[str],
        slots: int,
        config: ClientConfig,
        action_queue: "Queue[Action]",
        event_queue: "Queue[ActionEvent | STOP_LOOP_TYPE]",
        handle_kill: bool,
        debug: bool,
        labels: dict[str, str | int],
    ) -> None:
        self.name = name
        self.actions = actions
        self.slots = slots
        self.config = config
        self.action_queue = action_queue
        self.event_queue = event_queue
        self.debug = debug
        self.labels = labels
        self.handle_kill = handle_kill

        self.listener: ActionListener | None = None
        self.killing = False
        self.action_loop_task: asyncio.Task[None] | None = None
        self.event_send_loop_task: asyncio.Task[None] | None = None
        self.running_step_runs: dict[str, float] = {}
        self.step_action_events: set[
            asyncio.Task[UnaryUnaryCall[StepActionEvent, ActionEventResponse] | None]
        ] = set()
        self.group_key_action_events: set[
            asyncio.Task[
                UnaryUnaryCall[GroupKeyActionEvent, ActionEventResponse] | None
            ]
        ] = set()

        if self.debug:
            logger.setLevel(logging.DEBUG)

        self.client = Client(config=self.config, debug=self.debug)

        loop = asyncio.get_event_loop()
        loop.add_signal_handler(
            signal.SIGINT, lambda: asyncio.create_task(self.pause_task_assignment())
        )
        loop.add_signal_handler(
            signal.SIGTERM, lambda: asyncio.create_task(self.pause_task_assignment())
        )
        loop.add_signal_handler(
            signal.SIGQUIT, lambda: asyncio.create_task(self.exit_gracefully())
        )

    async def pause_task_assignment(self) -> None:
        if self.listener is None:
            raise ValueError("listener not started")

        await self.client.workers.aio_update(
            worker_id=self.listener.worker_id,
            opts=UpdateWorkerRequest(isPaused=True),
        )

    async def start(self, retry_attempt: int = 0) -> None:
        if retry_attempt > 5:
            logger.error("could not start action listener")
            return

        logger.debug(f"starting action listener: {self.name}")

        try:
            self.dispatcher_client = DispatcherClient(self.config)

            self.listener = await self.dispatcher_client.get_action_listener(
                GetActionListenerRequest(
                    worker_name=self.name,
                    services=["default"],
                    actions=self.actions,
                    slots=self.slots,
                    raw_labels=self.labels,
                )
            )

            logger.debug(f"acquired action listener: {self.listener.worker_id}")
        except grpc.RpcError as rpc_error:
            logger.error(f"could not start action listener: {rpc_error}")
            return

        # Start both loops as background tasks
        self.action_loop_task = asyncio.create_task(self.start_action_loop())
        self.event_send_loop_task = asyncio.create_task(self.start_event_send_loop())
        self.blocked_main_loop = asyncio.create_task(self.start_blocked_main_loop())

    # TODO move event methods to separate class
    async def _get_event(self) -> ActionEvent | STOP_LOOP_TYPE:
        loop = asyncio.get_running_loop()
        return await loop.run_in_executor(None, self.event_queue.get)

    async def start_event_send_loop(self) -> None:
        while True:
            event = await self._get_event()
            if event == STOP_LOOP:
                logger.debug("stopping event send loop...")
                break

            logger.debug(f"tx: event: {event.action.action_id}/{event.type}")
            t = asyncio.create_task(self.send_event(event))
            self.step_action_events.add(t)
            t.add_done_callback(lambda t: self.step_action_events.discard(t))

    async def start_blocked_main_loop(self) -> None:
        threshold = 1
        while not self.killing:
            count = 0
            for start_time in self.running_step_runs.values():
                diff = self.now() - start_time
                if diff > threshold:
                    count += 1

            if count > 0:
                logger.warning(f"{BLOCKED_THREAD_WARNING}: Waiting Steps {count}")
            await asyncio.sleep(1)

    async def send_event(self, event: ActionEvent, retry_attempt: int = 1) -> None:
        try:
            match event.action.action_type:
                # FIXME: all events sent from an execution of a function are of type ActionType.START_STEP_RUN since
                # the action is re-used. We should change this.
                case ActionType.START_STEP_RUN:
                    # TODO right now we're sending two start_step_run events
                    # one on the action loop and one on the event loop
                    # ideally we change the first to an ack to set the time
                    if event.type == STEP_EVENT_TYPE_STARTED:
                        if event.action.step_run_id in self.running_step_runs:
                            diff = (
                                self.now()
                                - self.running_step_runs[event.action.step_run_id]
                            )
                            if diff > 0.1:
                                logger.warning(
                                    f"{BLOCKED_THREAD_WARNING}: time to start: {diff}s"
                                )
                            else:
                                logger.debug(f"start time: {diff}")
                            del self.running_step_runs[event.action.step_run_id]
                        else:
                            self.running_step_runs[event.action.step_run_id] = (
                                self.now()
                            )

                    send_started_event_task = asyncio.create_task(
                        self.dispatcher_client.send_step_action_event(
                            event.action,
                            event.type,
                            event.payload,
                            event.should_not_retry,
                        )
                    )

                    self.step_action_events.add(send_started_event_task)
                    send_started_event_task.add_done_callback(
                        lambda t: self.step_action_events.discard(t)
                    )
                case ActionType.CANCEL_STEP_RUN:
                    logger.debug("unimplemented event send")
                case ActionType.START_GET_GROUP_KEY:
                    get_group_key_task = asyncio.create_task(
                        self.dispatcher_client.send_group_key_action_event(
                            event.action, event.type, event.payload
                        )
                    )
                    self.group_key_action_events.add(get_group_key_task)
                    get_group_key_task.add_done_callback(
                        lambda t: self.group_key_action_events.discard(t)
                    )
                case _:
                    logger.error("unknown action type for event send")
        except Exception as e:
            logger.error(
                f"could not send action event ({retry_attempt}/{ACTION_EVENT_RETRY_COUNT}): {e}"
            )
            if retry_attempt <= ACTION_EVENT_RETRY_COUNT:
                await exp_backoff_sleep(retry_attempt, 1)
                await self.send_event(event, retry_attempt + 1)

    def now(self) -> float:
        return time.time()

    async def start_action_loop(self) -> None:
        if self.listener is None:
            raise ValueError("listener not started")

        try:
            async for action in self.listener:
                if action is None:
                    break

                ctx_step_run_id.set(action.step_run_id)
                ctx_workflow_run_id.set(action.workflow_run_id)
                ctx_worker_id.set(action.worker_id)
                ctx_action_key.set(action.key)

                # Process the action here
                match action.action_type:
                    case ActionType.START_STEP_RUN:
                        self.event_queue.put(
                            ActionEvent(
                                action=action,
                                type=STEP_EVENT_TYPE_STARTED,  # TODO ack type
                                payload="",
                                should_not_retry=False,
                            )
                        )
                        logger.info(
                            f"rx: start step run: {action.step_run_id}/{action.action_id}"
                        )

                        # TODO handle this case better...
                        if action.step_run_id in self.running_step_runs:
                            logger.warning(
                                f"step run already running: {action.step_run_id}"
                            )

                    case ActionType.CANCEL_STEP_RUN:
                        logger.info(f"rx: cancel step run: {action.step_run_id}")
                    case ActionType.START_GET_GROUP_KEY:
                        self.event_queue.put(
                            ActionEvent(
                                action=action,
                                type=GROUP_KEY_EVENT_TYPE_STARTED,  # TODO ack type
                                payload="",
                                should_not_retry=False,
                            )
                        )
                        logger.info(
                            f"rx: start group key: {action.get_group_key_run_id}"
                        )
                    case _:
                        logger.error(
                            f"rx: unknown action type ({action.action_type}): {action.action_type}"
                        )
                try:
                    self.action_queue.put(action)
                except Exception as e:
                    logger.error(f"error putting action: {e}")

        except Exception as e:
            logger.error(f"error in action loop: {e}")
        finally:
            logger.info("action loop closed")
            if not self.killing:
                await self.exit_gracefully()

    async def cleanup(self) -> None:
        self.killing = True

        if self.listener is not None:
            self.listener.cleanup()

        self.event_queue.put(STOP_LOOP)

    async def exit_gracefully(self) -> None:
        if self.listener:
            self.listener.stop_signal = True

        await self.pause_task_assignment()

        if self.killing:
            return

        logger.debug("closing action listener...")

        await self.cleanup()

        while not self.event_queue.empty():
            pass

        logger.info("action listener closed")

    def exit_forcefully(self) -> None:
        asyncio.run(self.cleanup())
        logger.debug("forcefully closing listener...")


def worker_action_listener_process(*args: Any, **kwargs: Any) -> None:
    async def run() -> None:
        process = WorkerActionListenerProcess(*args, **kwargs)
        await process.start()
        # Keep the process running
        while not process.killing:  # noqa: ASYNC110
            await asyncio.sleep(0.1)

    asyncio.run(run())
