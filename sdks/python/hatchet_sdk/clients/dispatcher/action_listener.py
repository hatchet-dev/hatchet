import asyncio
import json
import time
from collections.abc import AsyncGenerator
from typing import TYPE_CHECKING, cast

import grpc
import grpc.aio
from pydantic import BaseModel, ConfigDict, Field, model_validator

from hatchet_sdk.clients.event_ts import (
    ThreadSafeEvent,
    UnexpectedEOF,
    read_with_interrupt,
)
from hatchet_sdk.clients.events import proto_timestamp_now
from hatchet_sdk.clients.listeners.run_event_listener import (
    DEFAULT_ACTION_LISTENER_RETRY_INTERVAL,
)
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.dispatcher_pb2 import ActionType as ActionTypeProto
from hatchet_sdk.contracts.dispatcher_pb2 import (
    AssignedAction,
    HeartbeatRequest,
    WorkerLabels,
    WorkerListenRequest,
    WorkerUnsubscribeRequest,
)
from hatchet_sdk.contracts.dispatcher_pb2_grpc import DispatcherStub
from hatchet_sdk.logger import logger
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.runnables.action import Action, ActionPayload, ActionType
from hatchet_sdk.utils.backoff import exp_backoff_sleep
from hatchet_sdk.utils.proto_enums import convert_proto_enum_to_python
from hatchet_sdk.utils.typing import JSONSerializableMapping

if TYPE_CHECKING:
    from hatchet_sdk.config import ClientConfig


DEFAULT_ACTION_TIMEOUT = 600  # seconds
DEFAULT_ACTION_LISTENER_RETRY_COUNT = 15


class GetActionListenerRequest(BaseModel):
    model_config = ConfigDict(arbitrary_types_allowed=True)

    worker_name: str
    services: list[str]
    actions: list[str]
    slots: int
    raw_labels: dict[str, str | int] = Field(default_factory=dict)

    labels: dict[str, WorkerLabels] = Field(default_factory=dict)

    @model_validator(mode="after")
    def validate_labels(self) -> "GetActionListenerRequest":
        self.labels = {}

        for key, value in self.raw_labels.items():
            if isinstance(value, int):
                self.labels[key] = WorkerLabels(intValue=value)
            else:
                self.labels[key] = WorkerLabels(strValue=str(value))

        return self


def parse_additional_metadata(additional_metadata: str) -> JSONSerializableMapping:
    try:
        return cast(
            JSONSerializableMapping,
            json.loads(additional_metadata),
        )
    except json.JSONDecodeError:
        return {}


class ActionListener:
    def __init__(self, config: "ClientConfig", worker_id: str) -> None:
        self.config = config
        self.worker_id = worker_id

        self.aio_client = DispatcherStub(new_conn(self.config, True))
        self.token = self.config.token

        self.retries = 0
        self.last_heartbeat_succeeded = True
        self.time_last_hb_succeeded = 9999999999999.0
        self.last_connection_attempt = 0.0
        self.heartbeat_task: asyncio.Task[None] | None = None
        self.run_heartbeat = True
        self.listen_strategy = "v2"
        self.stop_signal = False
        self.missed_heartbeats = 0

    def is_healthy(self) -> bool:
        return self.last_heartbeat_succeeded

    async def heartbeat(self) -> None:
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

    async def start_heartbeater(self) -> None:
        if self.heartbeat_task is not None:
            return

        loop = asyncio.get_event_loop()

        self.heartbeat_task = loop.create_task(self.heartbeat())

    def __aiter__(self) -> AsyncGenerator[Action | None, None]:
        return self._generator()

    async def _generator(self) -> AsyncGenerator[Action | None, None]:
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

                    if listener is None:
                        continue

                    t = asyncio.create_task(
                        read_with_interrupt(listener, self.interrupt)
                    )
                    await self.interrupt.wait()

                    if not t.done():
                        logger.warning(
                            "Interrupted read_with_interrupt task of action listener"
                        )

                        t.cancel()
                        listener.cancel()

                        break

                    result = t.result()

                    if isinstance(result, UnexpectedEOF):
                        logger.debug("Handling EOF in Action Listener")
                        self.retries = self.retries + 1
                        break

                    self.retries = 0

                    assigned_action = result.data

                    try:
                        action_payload = (
                            ActionPayload()
                            if not assigned_action.actionPayload
                            else ActionPayload.model_validate_json(
                                assigned_action.actionPayload
                            )
                        )
                    except (ValueError, json.JSONDecodeError) as e:
                        logger.error(f"Error decoding payload: {e}")

                        action_payload = ActionPayload()

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
                        action_type=convert_proto_enum_to_python(
                            assigned_action.actionType,
                            ActionType,
                            ActionTypeProto,
                        ),
                        retry_count=assigned_action.retryCount,
                        additional_metadata=parse_additional_metadata(
                            assigned_action.additional_metadata
                        ),
                        child_workflow_index=assigned_action.child_workflow_index,
                        child_workflow_key=assigned_action.child_workflow_key,
                        parent_workflow_run_id=assigned_action.parent_workflow_run_id,
                        priority=assigned_action.priority,
                        workflow_version_id=assigned_action.workflowVersionId,
                        workflow_id=assigned_action.workflowId,
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

    async def get_listen_client(
        self,
    ) -> grpc.aio.UnaryStreamCall[WorkerListenRequest, AssignedAction]:
        current_time = time.time()

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
        if self.retries >= 1:
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

        return cast(
            grpc.aio.UnaryStreamCall[WorkerListenRequest, AssignedAction], listener
        )

    def cleanup(self) -> None:
        self.run_heartbeat = False
        if self.heartbeat_task is not None:
            self.heartbeat_task.cancel()

        try:
            self.unregister()
        except Exception as e:
            logger.error(f"failed to unregister: {e}")

        if self.interrupt:  # type: ignore[truthy-bool]
            self.interrupt.set()

    def unregister(self) -> WorkerUnsubscribeRequest:
        self.run_heartbeat = False

        if self.heartbeat_task is not None:
            self.heartbeat_task.cancel()

        try:
            req = self.aio_client.Unsubscribe(
                WorkerUnsubscribeRequest(workerId=self.worker_id),
                timeout=5,
                metadata=get_metadata(self.token),
            )

            if self.interrupt is not None:
                self.interrupt.set()

            return cast(WorkerUnsubscribeRequest, req)
        except grpc.RpcError as e:
            raise Exception("Failed to unsubscribe") from e
