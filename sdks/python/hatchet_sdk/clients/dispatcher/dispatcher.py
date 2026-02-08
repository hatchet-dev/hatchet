import platform
from importlib.metadata import version
from sys import version_info
from typing import cast

import grpc.aio
from google.protobuf.timestamp_pb2 import Timestamp

from hatchet_sdk.clients.dispatcher.action_listener import (
    ActionListener,
    GetActionListenerRequest,
)
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.dispatcher_pb2 import (
    SDKS,
    STEP_EVENT_TYPE_COMPLETED,
    STEP_EVENT_TYPE_FAILED,
    ActionEventResponse,
    OverridesData,
    RefreshTimeoutRequest,
    ReleaseSlotRequest,
    RuntimeInfo,
    StepActionEvent,
    StepActionEventType,
    UpsertWorkerLabelsRequest,
    WorkerLabels,
    WorkerRegisterRequest,
    WorkerRegisterResponse,
)
from hatchet_sdk.contracts.dispatcher_pb2_grpc import DispatcherStub
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.runnables.action import Action

DEFAULT_REGISTER_TIMEOUT = 30


class DispatcherClient:
    def __init__(self, config: ClientConfig):
        self.token = config.token
        self.config = config

        ## IMPORTANT: This needs to be created lazily so we don't require
        ## an event loop to instantiate the client.
        self.aio_client: DispatcherStub | None = None
        self.client: DispatcherStub | None = None

    def _get_or_create_client(self) -> DispatcherStub:
        if self.client is None:
            conn = new_conn(self.config, False)
            self.client = DispatcherStub(conn)

        return self.client

    async def get_action_listener(
        self, req: GetActionListenerRequest
    ) -> ActionListener:
        if not self.aio_client:
            aio_conn = new_conn(self.config, True)
            self.aio_client = DispatcherStub(aio_conn)

        # Override labels with the preset labels
        preset_labels = self.config.worker_preset_labels

        for key, value in preset_labels.items():
            req.labels[key] = WorkerLabels(str_value=str(value))

        response = cast(
            WorkerRegisterResponse,
            # fixme: figure out how to get typing right here
            await self.aio_client.Register(
                WorkerRegisterRequest(
                    worker_name=req.worker_name,
                    actions=req.actions,
                    services=req.services,
                    labels=req.labels,
                    slot_config=req.slot_config,
                    runtime_info=RuntimeInfo(
                        sdk_version=version("hatchet_sdk"),
                        language=SDKS.PYTHON,
                        language_version=f"{version_info.major}.{version_info.minor}.{version_info.micro}",
                        os=platform.system().lower(),
                    ),
                ),
                timeout=DEFAULT_REGISTER_TIMEOUT,
                metadata=get_metadata(self.token),
            ),
        )

        return ActionListener(self.config, response.worker_id)

    async def send_step_action_event(
        self,
        action: Action,
        event_type: StepActionEventType,
        payload: str | None,
        should_not_retry: bool,
    ) -> grpc.aio.UnaryUnaryCall[StepActionEvent, ActionEventResponse] | None:
        try:
            return await self._try_send_step_action_event(
                action, event_type, payload, should_not_retry
            )
        except Exception as e:
            # for step action events, send a failure event when we cannot send the completed event
            if event_type in (STEP_EVENT_TYPE_COMPLETED, STEP_EVENT_TYPE_FAILED):
                await self._try_send_step_action_event(
                    action,
                    STEP_EVENT_TYPE_FAILED,
                    "Failed to send finished event: " + str(e),
                    should_not_retry=True,
                )

            return None

    async def _try_send_step_action_event(
        self,
        action: Action,
        event_type: StepActionEventType,
        payload: str | None,
        should_not_retry: bool,
    ) -> grpc.aio.UnaryUnaryCall[StepActionEvent, ActionEventResponse]:
        if not self.aio_client:
            aio_conn = new_conn(self.config, True)
            self.aio_client = DispatcherStub(aio_conn)

        event_timestamp = Timestamp()
        event_timestamp.GetCurrentTime()

        event = StepActionEvent(
            worker_id=action.worker_id,
            job_id=action.job_id,
            job_run_id=action.job_run_id,
            task_id=action.step_id,
            task_run_external_id=action.step_run_id,
            action_id=action.action_id,
            event_timestamp=event_timestamp,
            event_type=event_type,
            event_payload=payload,
            retry_count=action.retry_count,
            should_not_retry=should_not_retry,
        )

        send_step_action_event = tenacity_retry(
            self.aio_client.SendStepActionEvent, self.config.tenacity
        )

        return cast(
            grpc.aio.UnaryUnaryCall[StepActionEvent, ActionEventResponse],
            # fixme: figure out how to get typing right here
            await send_step_action_event(
                event,
                metadata=get_metadata(self.token),
            ),
        )

    def put_overrides_data(self, data: OverridesData) -> ActionEventResponse:
        client = self._get_or_create_client()

        return cast(
            ActionEventResponse,
            client.PutOverridesData(
                data,
                metadata=get_metadata(self.token),
            ),
        )

    def release_slot(self, step_run_id: str) -> None:
        client = self._get_or_create_client()

        client.ReleaseSlot(
            ReleaseSlotRequest(task_run_external_id=step_run_id),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=get_metadata(self.token),
        )

    def refresh_timeout(self, step_run_id: str, increment_by: str) -> None:
        client = self._get_or_create_client()

        client.RefreshTimeout(
            RefreshTimeoutRequest(
                task_run_external_id=step_run_id,
                increment_timeout_by=increment_by,
            ),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=get_metadata(self.token),
        )

    def upsert_worker_labels(
        self, worker_id: str | None, labels: dict[str, str | int]
    ) -> None:
        worker_labels = {}

        for key, value in labels.items():
            if isinstance(value, int):
                worker_labels[key] = WorkerLabels(int_value=value)
            else:
                worker_labels[key] = WorkerLabels(str_value=str(value))

        client = self._get_or_create_client()

        client.UpsertWorkerLabels(
            UpsertWorkerLabelsRequest(worker_id=worker_id, labels=worker_labels),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=get_metadata(self.token),
        )

    async def async_upsert_worker_labels(
        self,
        worker_id: str | None,
        labels: dict[str, str | int],
    ) -> None:
        if not self.aio_client:
            aio_conn = new_conn(self.config, True)
            self.aio_client = DispatcherStub(aio_conn)

        worker_labels = {}

        for key, value in labels.items():
            if isinstance(value, int):
                worker_labels[key] = WorkerLabels(int_value=value)
            else:
                worker_labels[key] = WorkerLabels(str_value=str(value))

        # fixme: figure out how to get typing right here
        await self.aio_client.UpsertWorkerLabels(
            UpsertWorkerLabelsRequest(worker_id=worker_id, labels=worker_labels),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=get_metadata(self.token),
        )
