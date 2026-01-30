import asyncio
import contextlib
import logging
import signal
import time
from dataclasses import dataclass
from datetime import timedelta
from enum import Enum
from multiprocessing import Queue
from typing import Any

import grpc
from aiohttp import web
from aiohttp.web_request import Request
from aiohttp.web_response import Response
from grpc.aio import UnaryUnaryCall
from prometheus_client import Gauge, generate_latest

from hatchet_sdk.client import Client
from hatchet_sdk.clients.dispatcher.action_listener import (
    ActionListener,
    GetActionListenerRequest,
)
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.rest.models.update_worker_request import UpdateWorkerRequest
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.contracts.dispatcher_pb2 import (
    STEP_EVENT_TYPE_STARTED,
    ActionEventResponse,
    StepActionEvent,
)
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.action import Action, ActionType
from hatchet_sdk.runnables.contextvars import (
    ctx_action_key,
    ctx_step_run_id,
    ctx_task_retry_count,
    ctx_worker_id,
    ctx_workflow_run_id,
)
from hatchet_sdk.utils.backoff import exp_backoff_sleep
from hatchet_sdk.utils.typing import STOP_LOOP, STOP_LOOP_TYPE

ACTION_EVENT_RETRY_COUNT = 5
STARTING_UNHEALTHY_AFTER_SECONDS = 10.0


class HealthStatus(str, Enum):
    STARTING = "STARTING"
    HEALTHY = "HEALTHY"
    UNHEALTHY = "UNHEALTHY"


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

        self._health_runner: web.AppRunner | None = None
        self._listener_health_gauge: Gauge | None = None
        self._event_loop_lag_gauge: Gauge | None = None
        self._event_loop_monitor_task: asyncio.Task[None] | None = None
        self._event_loop_last_lag_seconds: float = 0.0
        self._event_loop_blocked_since: float | None = None
        self._waiting_steps_blocked_since: float | None = None
        self._starting_since: float = time.time()

        self.listener: ActionListener | None = None
        self.killing = False
        self.action_loop_task: asyncio.Task[None] | None = None
        self.event_send_loop_task: asyncio.Task[None] | None = None
        self.running_step_runs: dict[str, float] = {}
        self.step_action_events: set[
            asyncio.Task[UnaryUnaryCall[StepActionEvent, ActionEventResponse] | None]
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

        if self.config.healthcheck.enabled:
            self._listener_health_gauge = Gauge(
                "hatchet_worker_listener_health",
                "Listener health (1 healthy, 0 unhealthy)",
            )
            self._event_loop_lag_gauge = Gauge(
                "hatchet_worker_event_loop_lag_seconds",
                "Event loop lag in seconds (listener process)",
            )

    async def _monitor_event_loop(self) -> None:
        # If the loop is blocked, this coroutine itself can't run; when it resumes,
        # we detect the lag by comparing elapsed time vs expected sleep.
        interval = 0.5

        while not self.killing:
            start = time.time()
            await asyncio.sleep(interval)
            elapsed = time.time() - start
            lag = max(0.0, elapsed - interval)
            # If the loop is "completely blocked" across multiple monitor ticks,
            # report a continuously increasing lag value (time since first detected block).
            if (
                timedelta(seconds=lag)
                >= self.config.healthcheck.event_loop_block_threshold_seconds
            ):
                if self._event_loop_blocked_since is None:
                    self._event_loop_blocked_since = start + interval
                self._event_loop_last_lag_seconds = max(
                    lag, time.time() - self._event_loop_blocked_since
                )
            else:
                self._event_loop_last_lag_seconds = lag

            if (
                timedelta(seconds=lag)
                < self.config.healthcheck.event_loop_block_threshold_seconds
            ):
                self._event_loop_blocked_since = None

    def _starting_timed_out(self) -> bool:
        return (time.time() - self._starting_since) > STARTING_UNHEALTHY_AFTER_SECONDS

    def _compute_health(self) -> HealthStatus:
        if self.killing:
            return HealthStatus.UNHEALTHY

        # If the event loop has been blocked longer than the configured threshold, report unhealthy.
        if (
            self._event_loop_blocked_since is not None
            and timedelta(seconds=(time.time() - self._event_loop_blocked_since))
            > self.config.healthcheck.event_loop_block_threshold_seconds
        ):
            return HealthStatus.UNHEALTHY

        # If steps have been waiting to start for longer than the threshold,
        # treat this as unhealthy as well (this is what triggers the
        # "Waiting Steps" blocked-loop warning).
        if (
            self._waiting_steps_blocked_since is not None
            and timedelta(seconds=(time.time() - self._waiting_steps_blocked_since))
            > self.config.healthcheck.event_loop_block_threshold_seconds
        ):
            return HealthStatus.UNHEALTHY

        if self.listener is None:
            if self._starting_timed_out():
                return HealthStatus.UNHEALTHY
            return HealthStatus.STARTING

        listener = self.listener

        # Avoid false positives before we have any listener connection attempts.
        last_attempt = listener.last_connection_attempt or 0.0
        if last_attempt <= 0:
            if self._starting_timed_out():
                return HealthStatus.UNHEALTHY
            return HealthStatus.STARTING

        if listener.listen_strategy == "v2":
            # Require at least one successful heartbeat.
            #
            # Note: the listener initializes `time_last_hb_succeeded` to a sentinel
            # value; only treat it as "real" after it's been updated to a timestamp
            # <= now.
            now = time.time()
            time_last_hb = listener.time_last_hb_succeeded or 0.0
            has_hb_success = 0.0 < time_last_hb <= now
            ok = bool(
                listener.heartbeat_task is not None
                and listener.last_heartbeat_succeeded
                and has_hb_success
            )
        else:
            # For v1 listen strategy (no heartbeater), treat "no retries" as healthy.
            ok = bool(listener.retries == 0)

        return HealthStatus.HEALTHY if ok else HealthStatus.UNHEALTHY

    async def _health_handler(self, request: Request) -> Response:
        status = self._compute_health()
        ok = status == HealthStatus.HEALTHY

        # Keep this response minimal because the endpoint is public.
        response = {"status": status.value}

        return web.json_response(response, status=200 if ok else 503)

    async def _metrics_handler(self, request: Request) -> Response:
        status = self._compute_health()
        ok = status == HealthStatus.HEALTHY

        if self._listener_health_gauge is not None:
            self._listener_health_gauge.set(1 if ok else 0)

        if self._event_loop_lag_gauge is not None:
            self._event_loop_lag_gauge.set(self._event_loop_last_lag_seconds)

        # Note: this is a local Prometheus endpoint for the worker process itself.
        return web.Response(body=generate_latest(), content_type="text/plain")

    async def start_health_server(self) -> None:
        if not self.config.healthcheck.enabled:
            return

        if self._health_runner is not None:
            return

        app = web.Application()
        app.add_routes(
            [
                web.get("/health", self._health_handler),
                web.get("/metrics", self._metrics_handler),
            ]
        )

        runner = web.AppRunner(app)

        try:
            await runner.setup()
            await web.TCPSite(
                runner,
                host=self.config.healthcheck.bind_address,
                port=self.config.healthcheck.port,
            ).start()
        except Exception:
            logger.exception("failed to start healthcheck server (listener process)")
            return

        self._health_runner = runner
        logger.info(
            f"healthcheck server (listener process) running on {self.config.healthcheck.bind_address}:{self.config.healthcheck.port}"
        )

        if self._event_loop_monitor_task is None:
            self._event_loop_monitor_task = asyncio.create_task(
                self._monitor_event_loop()
            )

    async def stop_health_server(self) -> None:
        if self._event_loop_monitor_task is not None:
            task = self._event_loop_monitor_task
            self._event_loop_monitor_task = None
            task.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await task

        if self._health_runner is None:
            return

        try:
            await self._health_runner.cleanup()
        except Exception:
            logger.exception("failed to stop healthcheck server (listener process)")
        finally:
            self._health_runner = None

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
        except grpc.RpcError:
            logger.exception("could not start action listener")
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
                if self._waiting_steps_blocked_since is None:
                    self._waiting_steps_blocked_since = time.time()
                blocked_for = time.time() - self._waiting_steps_blocked_since
                # Continuously increasing "lag length" while we're blocked waiting for steps to start.
                logger.warning(
                    f"{BLOCKED_THREAD_WARNING} Waiting Steps {count} blocked_for={blocked_for:.1f}s"
                )
            else:
                self._waiting_steps_blocked_since = None
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
                                    f"{BLOCKED_THREAD_WARNING} time to start: {diff}s"
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
                case _:
                    logger.error("unknown action type for event send")
        except Exception:
            logger.exception(
                f"could not send action event ({retry_attempt}/{ACTION_EVENT_RETRY_COUNT})"
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
                ctx_task_retry_count.set(action.retry_count)

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
                    case _:
                        logger.error(
                            f"rx: unknown action type ({action.action_type}): {action.action_type}"
                        )
                try:
                    self.action_queue.put(action)
                except Exception:
                    logger.exception("error putting action")

        except Exception:
            logger.exception("error in action loop")
        finally:
            logger.info("action loop closed")
            if not self.killing:
                await self.exit_gracefully()

    async def cleanup(self) -> None:
        self.killing = True

        await self.stop_health_server()

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


def worker_action_listener_process(
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
    async def run() -> None:
        process = WorkerActionListenerProcess(
            name=name,
            actions=actions,
            slots=slots,
            config=config,
            action_queue=action_queue,
            event_queue=event_queue,
            handle_kill=handle_kill,
            debug=debug,
            labels=labels,
        )
        await process.start_health_server()
        await process.start()
        # Keep the process running
        while not process.killing:  # noqa: ASYNC110
            await asyncio.sleep(0.1)

    asyncio.run(run())
