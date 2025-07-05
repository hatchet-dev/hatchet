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
    STEP_EVENT_TYPE_COMPLETED,
    STEP_EVENT_TYPE_FAILED,
    ActionEventResponse,
    GroupKeyActionEvent,
    GroupKeyActionEventType,
    OverridesData,
    RefreshTimeoutRequest,
    ReleaseSlotRequest,
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
            req.labels[key] = WorkerLabels(strValue=str(value))

        response = cast(
            WorkerRegisterResponse,
            await self.aio_client.Register(
                WorkerRegisterRequest(
                    workerName=req.worker_name,
                    actions=req.actions,
                    services=req.services,
                    maxRuns=req.slots,
                    labels=req.labels,
                ),
                timeout=DEFAULT_REGISTER_TIMEOUT,
                metadata=get_metadata(self.token),
            ),
        )

        return ActionListener(self.config, response.workerId)

    async def send_step_action_event(
        self,
        action: Action,
        event_type: StepActionEventType,
        payload: str,
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

    @tenacity_retry
    async def _try_send_step_action_event(
        self,
        action: Action,
        event_type: StepActionEventType,
        payload: str,
        should_not_retry: bool,
    ) -> grpc.aio.UnaryUnaryCall[StepActionEvent, ActionEventResponse]:
        if not self.aio_client:
            aio_conn = new_conn(self.config, True)
            self.aio_client = DispatcherStub(aio_conn)

        event_timestamp = Timestamp()
        event_timestamp.GetCurrentTime()

        event = StepActionEvent(
            workerId=action.worker_id,
            jobId=action.job_id,
            jobRunId=action.job_run_id,
            stepId=action.step_id,
            stepRunId=action.step_run_id,
            actionId=action.action_id,
            eventTimestamp=event_timestamp,
            eventType=event_type,
            eventPayload=payload,
            retryCount=action.retry_count,
            shouldNotRetry=should_not_retry,
        )

        return cast(
            grpc.aio.UnaryUnaryCall[StepActionEvent, ActionEventResponse],
            await self.aio_client.SendStepActionEvent(
                event,
                metadata=get_metadata(self.token),
            ),
        )

    async def send_group_key_action_event(
        self, action: Action, event_type: GroupKeyActionEventType, payload: str
    ) -> grpc.aio.UnaryUnaryCall[GroupKeyActionEvent, ActionEventResponse]:
        if not self.aio_client:
            aio_conn = new_conn(self.config, True)
            self.aio_client = DispatcherStub(aio_conn)

        event_timestamp = Timestamp()
        event_timestamp.GetCurrentTime()

        event = GroupKeyActionEvent(
            workerId=action.worker_id,
            workflowRunId=action.workflow_run_id,
            getGroupKeyRunId=action.get_group_key_run_id,
            actionId=action.action_id,
            eventTimestamp=event_timestamp,
            eventType=event_type,
            eventPayload=payload,
        )

        return cast(
            grpc.aio.UnaryUnaryCall[GroupKeyActionEvent, ActionEventResponse],
            await self.aio_client.SendGroupKeyActionEvent(
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
            ReleaseSlotRequest(stepRunId=step_run_id),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=get_metadata(self.token),
        )

    def refresh_timeout(self, step_run_id: str, increment_by: str) -> None:
        client = self._get_or_create_client()

        client.RefreshTimeout(
            RefreshTimeoutRequest(
                stepRunId=step_run_id,
                incrementTimeoutBy=increment_by,
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
                worker_labels[key] = WorkerLabels(intValue=value)
            else:
                worker_labels[key] = WorkerLabels(strValue=str(value))

        client = self._get_or_create_client()

        client.UpsertWorkerLabels(
            UpsertWorkerLabelsRequest(workerId=worker_id, labels=worker_labels),
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
                worker_labels[key] = WorkerLabels(intValue=value)
            else:
                worker_labels[key] = WorkerLabels(strValue=str(value))

        await self.aio_client.UpsertWorkerLabels(
            UpsertWorkerLabelsRequest(workerId=worker_id, labels=worker_labels),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=get_metadata(self.token),
        )
