import asyncio
import json
import time
from dataclasses import dataclass, field
from typing import Any, AsyncGenerator, List, Optional

import grpc
from grpc._cython import cygrpc

from hatchet_sdk.contracts.dispatcher_pb2 import (
    ActionType,
    AssignedAction,
    HeartbeatRequest,
    WorkerLabels,
    WorkerListenRequest,
    WorkerUnsubscribeRequest,
)
from hatchet_sdk.contracts.dispatcher_pb2_grpc import DispatcherStub
from hatchet_sdk.logger import logger
from hatchet_sdk.v0.clients.event_ts import ThreadSafeEvent, read_with_interrupt
from hatchet_sdk.v0.clients.run_event_listener import (
    DEFAULT_ACTION_LISTENER_RETRY_INTERVAL,
)
from hatchet_sdk.v0.connection import new_conn
from hatchet_sdk.v0.utils.backoff import exp_backoff_sleep

from ...loader import ClientConfig
from ...metadata import get_metadata
from ..events import proto_timestamp_now

DEFAULT_ACTION_TIMEOUT = 600  # seconds


DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5  # seconds
DEFAULT_ACTION_LISTENER_RETRY_COUNT = 15


@dataclass
class GetActionListenerRequest:
    worker_name: str
    services: List[str]
    actions: List[str]
    max_runs: Optional[int] = None
    _labels: dict[str, str | int] = field(default_factory=dict)

    labels: dict[str, WorkerLabels] = field(init=False)

    def __post_init__(self):
        self.labels = {}

        for key, value in self._labels.items():
            if isinstance(value, int):
                self.labels[key] = WorkerLabels(intValue=value)
            else:
                self.labels[key] = WorkerLabels(strValue=str(value))


@dataclass
class Action:
    worker_id: str
    tenant_id: str
    workflow_run_id: str
    get_group_key_run_id: str
    job_id: str
    job_name: str
    job_run_id: str
    step_id: str
    step_run_id: str
    action_id: str
    action_payload: str
    action_type: ActionType
    retry_count: int
    additional_metadata: dict[str, str] | None = None

    child_workflow_index: int | None = None
    child_workflow_key: str | None = None
    parent_workflow_run_id: str | None = None

    def __post_init__(self):
        if isinstance(self.additional_metadata, str) and self.additional_metadata != "":
            try:
                self.additional_metadata = json.loads(self.additional_metadata)
            except json.JSONDecodeError:
                # If JSON decoding fails, keep the original string
                pass

        # Ensure additional_metadata is always a dictionary
        if not isinstance(self.additional_metadata, dict):
            self.additional_metadata = {}

    @property
    def otel_attributes(self) -> dict[str, str | int]:
        try:
            payload_str = json.dumps(self.action_payload, default=str)
        except Exception:
            payload_str = str(self.action_payload)

        attrs: dict[str, str | int | None] = {
            "hatchet.tenant_id": self.tenant_id,
            "hatchet.worker_id": self.worker_id,
            "hatchet.workflow_run_id": self.workflow_run_id,
            "hatchet.step_id": self.step_id,
            "hatchet.step_run_id": self.step_run_id,
            "hatchet.retry_count": self.retry_count,
            "hatchet.parent_workflow_run_id": self.parent_workflow_run_id,
            "hatchet.child_workflow_index": self.child_workflow_index,
            "hatchet.child_workflow_key": self.child_workflow_key,
            "hatchet.action_payload": payload_str,
            "hatchet.workflow_name": self.job_name,
            "hatchet.action_name": self.action_id,
            "hatchet.get_group_key_run_id": self.get_group_key_run_id,
        }

        return {k: v for k, v in attrs.items() if v}


START_STEP_RUN = 0
CANCEL_STEP_RUN = 1
START_GET_GROUP_KEY = 2


@dataclass
class ActionListener:
    config: ClientConfig
    worker_id: str

    client: DispatcherStub = field(init=False)
    aio_client: DispatcherStub = field(init=False)
    token: str = field(init=False)
    retries: int = field(default=0, init=False)
    last_connection_attempt: float = field(default=0, init=False)
    last_heartbeat_succeeded: bool = field(default=True, init=False)
    time_last_hb_succeeded: float = field(default=9999999999999, init=False)
    heartbeat_task: Optional[asyncio.Task] = field(default=None, init=False)
    run_heartbeat: bool = field(default=True, init=False)
    listen_strategy: str = field(default="v2", init=False)
    stop_signal: bool = field(default=False, init=False)

    missed_heartbeats: int = field(default=0, init=False)

    def __post_init__(self):
        self.client = DispatcherStub(new_conn(self.config))
        self.aio_client = DispatcherStub(new_conn(self.config, True))
        self.token = self.config.token

    def is_healthy(self):
        return self.last_heartbeat_succeeded

    async def heartbeat(self):
        # send a heartbeat every 4 seconds
        heartbeat_delay = 4

        while True:
            if not self.run_heartbeat:
                break

            try:
                logger.debug("sending heartbeat")
                await self.aio_client.Heartbeat(
                    HeartbeatRequest(
                        workerId=self.worker_id,
                        heartbeatAt=proto_timestamp_now(),
                    ),
                    timeout=5,
                    metadata=get_metadata(self.token),
                )

                if self.last_heartbeat_succeeded is False:
                    logger.info("listener established")

                now = time.time()
                diff = now - self.time_last_hb_succeeded
                if diff > heartbeat_delay + 1:
                    logger.warn(
                        f"time since last successful heartbeat: {diff:.2f}s, expects {heartbeat_delay}s"
                    )

                self.last_heartbeat_succeeded = True
                self.time_last_hb_succeeded = now
                self.missed_heartbeats = 0
            except grpc.RpcError as e:
                self.missed_heartbeats = self.missed_heartbeats + 1
                self.last_heartbeat_succeeded = False

                if (
                    e.code() == grpc.StatusCode.UNAVAILABLE
                    or e.code() == grpc.StatusCode.FAILED_PRECONDITION
                ):
                    # todo case on "recvmsg:Connection reset by peer" for updates?
                    if self.missed_heartbeats >= 3:
                        # we don't reraise the error here, as we don't want to stop the heartbeat thread
                        logger.error(
                            f"⛔️ failed heartbeat ({self.missed_heartbeats}): {e.details()}"
                        )
                    elif self.missed_heartbeats > 1:
                        logger.warning(
                            f"failed to send heartbeat ({self.missed_heartbeats}): {e.details()}"
                        )
                else:
                    logger.error(f"failed to send heartbeat: {e}")

                if self.interrupt is not None:
                    self.interrupt.set()

                if e.code() == grpc.StatusCode.UNIMPLEMENTED:
                    break
            await asyncio.sleep(heartbeat_delay)

    async def start_heartbeater(self):
        if self.heartbeat_task is not None:
            return

        try:
            loop = asyncio.get_event_loop()
        except RuntimeError as e:
            if str(e).startswith("There is no current event loop in thread"):
                loop = asyncio.new_event_loop()
                asyncio.set_event_loop(loop)
            else:
                raise e
        self.heartbeat_task = loop.create_task(self.heartbeat())

    def __aiter__(self):
        return self._generator()

    async def _generator(self) -> AsyncGenerator[Action, None]:
        listener = None

        while not self.stop_signal:
            if listener is not None:
                listener.cancel()

            try:
                listener = await self.get_listen_client()
            except Exception:
                logger.info("closing action listener loop")
                yield None

            try:
                while not self.stop_signal:
                    self.interrupt = ThreadSafeEvent()
                    t = asyncio.create_task(
                        read_with_interrupt(listener, self.interrupt)
                    )
                    await self.interrupt.wait()

                    if not t.done():
                        # print a warning
                        logger.warning(
                            "Interrupted read_with_interrupt task of action listener"
                        )

                        t.cancel()
                        listener.cancel()
                        break

                    assigned_action = t.result()

                    if assigned_action is cygrpc.EOF:
                        self.retries = self.retries + 1
                        break

                    self.retries = 0
                    assigned_action: AssignedAction

                    # Process the received action
                    action_type = self.map_action_type(assigned_action.actionType)

                    if (
                        assigned_action.actionPayload is None
                        or assigned_action.actionPayload == ""
                    ):
                        action_payload = None
                    else:
                        action_payload = self.parse_action_payload(
                            assigned_action.actionPayload
                        )

                    action = Action(
                        tenant_id=assigned_action.tenantId,
                        worker_id=self.worker_id,
                        workflow_run_id=assigned_action.workflowRunId,
                        get_group_key_run_id=assigned_action.getGroupKeyRunId,
                        job_id=assigned_action.jobId,
                        job_name=assigned_action.jobName,
                        job_run_id=assigned_action.jobRunId,
                        step_id=assigned_action.stepId,
                        step_run_id=assigned_action.stepRunId,
                        action_id=assigned_action.actionId,
                        action_payload=action_payload,
                        action_type=action_type,
                        retry_count=assigned_action.retryCount,
                        additional_metadata=assigned_action.additional_metadata,
                        child_workflow_index=assigned_action.child_workflow_index,
                        child_workflow_key=assigned_action.child_workflow_key,
                        parent_workflow_run_id=assigned_action.parent_workflow_run_id,
                    )

                    yield action
            except grpc.RpcError as e:
                self.last_heartbeat_succeeded = False

                # Handle different types of errors
                if e.code() == grpc.StatusCode.CANCELLED:
                    # Context cancelled, unsubscribe and close
                    logger.debug("Context cancelled, closing listener")
                elif e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
                    logger.info("Deadline exceeded, retrying subscription")
                elif (
                    self.listen_strategy == "v2"
                    and e.code() == grpc.StatusCode.UNIMPLEMENTED
                ):
                    # ListenV2 is not available, fallback to Listen
                    self.listen_strategy = "v1"
                    self.run_heartbeat = False
                    logger.info("ListenV2 not available, falling back to Listen")
                else:
                    # TODO retry
                    if e.code() == grpc.StatusCode.UNAVAILABLE:
                        logger.error(f"action listener error: {e.details()}")
                    else:
                        # Unknown error, report and break
                        logger.error(f"action listener error: {e}")

                    self.retries = self.retries + 1

    def parse_action_payload(self, payload: str):
        try:
            payload_data = json.loads(payload)
        except json.JSONDecodeError as e:
            raise ValueError(f"Error decoding payload: {e}")
        return payload_data

    def map_action_type(self, action_type):
        if action_type == ActionType.START_STEP_RUN:
            return START_STEP_RUN
        elif action_type == ActionType.CANCEL_STEP_RUN:
            return CANCEL_STEP_RUN
        elif action_type == ActionType.START_GET_GROUP_KEY:
            return START_GET_GROUP_KEY
        else:
            # logger.error(f"Unknown action type: {action_type}")
            return None

    async def get_listen_client(self):
        current_time = int(time.time())

        if (
            current_time - self.last_connection_attempt
            > DEFAULT_ACTION_LISTENER_RETRY_INTERVAL
        ):
            # reset retries if last connection was long lived
            self.retries = 0

        if self.retries > DEFAULT_ACTION_LISTENER_RETRY_COUNT:
            # TODO this is the problem case...
            logger.error(
                f"could not establish action listener connection after {DEFAULT_ACTION_LISTENER_RETRY_COUNT} retries"
            )
            self.run_heartbeat = False
            raise Exception("retry_exhausted")
        elif self.retries >= 1:
            # logger.info
            # if we are retrying, we wait for a bit. this should eventually be replaced with exp backoff + jitter
            await exp_backoff_sleep(
                self.retries, DEFAULT_ACTION_LISTENER_RETRY_INTERVAL
            )

            logger.info(
                f"action listener connection interrupted, retrying... ({self.retries}/{DEFAULT_ACTION_LISTENER_RETRY_COUNT})"
            )

        self.aio_client = DispatcherStub(new_conn(self.config, True))

        if self.listen_strategy == "v2":
            # we should await for the listener to be established before
            # starting the heartbeater
            listener = self.aio_client.ListenV2(
                WorkerListenRequest(workerId=self.worker_id),
                timeout=self.config.listener_v2_timeout,
                metadata=get_metadata(self.token),
            )
            await self.start_heartbeater()
        else:
            # if ListenV2 is not available, fallback to Listen
            listener = self.aio_client.Listen(
                WorkerListenRequest(workerId=self.worker_id),
                timeout=DEFAULT_ACTION_TIMEOUT,
                metadata=get_metadata(self.token),
            )

        self.last_connection_attempt = current_time

        return listener

    def cleanup(self):
        self.run_heartbeat = False
        self.heartbeat_task.cancel()

        try:
            self.unregister()
        except Exception as e:
            logger.error(f"failed to unregister: {e}")

        if self.interrupt:
            self.interrupt.set()

    def unregister(self):
        self.run_heartbeat = False
        self.heartbeat_task.cancel()

        try:
            req = self.aio_client.Unsubscribe(
                WorkerUnsubscribeRequest(workerId=self.worker_id),
                timeout=5,
                metadata=get_metadata(self.token),
            )
            if self.interrupt is not None:
                self.interrupt.set()
            return req
        except grpc.RpcError as e:
            raise Exception(f"Failed to unsubscribe: {e}")
