import platform
from importlib.metadata import version
from sys import version_info
from typing import cast

import grpc.aio
import tenacity
from google.protobuf.timestamp_pb2 import Timestamp

from hatchet_sdk.clients.dispatcher.action_listener import ActionListener
from hatchet_sdk.clients.rest.tenacity_utils import (
    tenacity_alert_retry,
    tenacity_retry,
    tenacity_should_retry,
)
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.dispatcher_pb2 import (
    SDKS,
    ActionEventResponse,
    GetVersionRequest,
    GetVersionResponse,
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
from hatchet_sdk.runnables.action import Action
from hatchet_sdk.types.labels import WorkerLabel
from hatchet_sdk.utils.api_auth import create_authorization_header

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
        self,
        worker_name: str,
        services: list[str],
        actions: list[str],
        slot_config: dict[str, int],
        labels: list[WorkerLabel],
    ) -> ActionListener:
        if not self.aio_client:
            aio_conn = new_conn(self.config, True)
            self.aio_client = DispatcherStub(aio_conn)

        proto_labels: dict[str, WorkerLabels] = {
            label.key: label.to_proto() for label in labels if label.key is not None
        }
        for key, value in self.config.worker_preset_labels.items():
            proto_labels[key] = WorkerLabels(str_value=str(value))

        response = cast(
            WorkerRegisterResponse,
            # fixme: figure out how to get typing right here
            await self.aio_client.Register(  # type: ignore[misc]
                WorkerRegisterRequest(
                    worker_name=worker_name,
                    actions=actions,
                    services=services,
                    labels=proto_labels,
                    slot_config=slot_config,
                    runtime_info=RuntimeInfo(
                        sdk_version=version("hatchet_sdk"),
                        language=SDKS.PYTHON,
                        language_version=f"{version_info.major}.{version_info.minor}.{version_info.micro}",
                        os=platform.system().lower(),
                    ),
                ),
                timeout=DEFAULT_REGISTER_TIMEOUT,
                metadata=create_authorization_header(self.token),
            ),
        )

        return ActionListener(self.config, response.worker_id)

    @tenacity.retry(
        reraise=True,
        wait=tenacity.wait_exponential_jitter(initial=0.5, max=5),
        stop=tenacity.stop_after_attempt(3),
        before_sleep=tenacity_alert_retry,
        retry=tenacity.retry_if_exception(tenacity_should_retry),
    )
    async def get_version(self) -> str | None:
        """Call GetVersion RPC. Returns the engine semantic version string,
        or ``None`` if the engine is too old to support GetVersion.

        Retries transient gRPC errors up to 3 times with exponential backoff.
        """
        if not self.aio_client:
            aio_conn = new_conn(self.config, True)
            self.aio_client = DispatcherStub(aio_conn)

        try:
            response = cast(
                GetVersionResponse,
                await self.aio_client.GetVersion(  # type: ignore[misc]
                    GetVersionRequest(),
                    timeout=DEFAULT_REGISTER_TIMEOUT,
                    metadata=create_authorization_header(self.token),
                ),
            )
        except grpc.RpcError as e:
            if e.code() == grpc.StatusCode.UNIMPLEMENTED:
                return None
            raise

        return response.version

    async def send_step_action_event(
        self,
        action: Action,
        event_type: StepActionEventType,
        payload: str | None,
        should_not_retry: bool,
    ) -> grpc.aio.UnaryUnaryCall[StepActionEvent, ActionEventResponse] | None:
        return await self._try_send_step_action_event(
            action, event_type, payload, should_not_retry
        )

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
            await send_step_action_event(  # type: ignore[misc]
                event,
                metadata=create_authorization_header(self.token),
            ),
        )

    def put_overrides_data(self, data: OverridesData) -> ActionEventResponse:
        client = self._get_or_create_client()

        return cast(
            ActionEventResponse,
            client.PutOverridesData(
                data,
                metadata=create_authorization_header(self.token),
            ),
        )

    def release_slot(self, step_run_id: str) -> None:
        client = self._get_or_create_client()

        client.ReleaseSlot(
            ReleaseSlotRequest(task_run_external_id=step_run_id),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=create_authorization_header(self.token),
        )

    def refresh_timeout(self, step_run_id: str, increment_by: str) -> None:
        client = self._get_or_create_client()

        client.RefreshTimeout(
            RefreshTimeoutRequest(
                task_run_external_id=step_run_id,
                increment_timeout_by=increment_by,
            ),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=create_authorization_header(self.token),
        )

    def upsert_worker_labels(
        self, worker_id: str | None, labels: list[WorkerLabel]
    ) -> None:
        client = self._get_or_create_client()

        client.UpsertWorkerLabels(
            UpsertWorkerLabelsRequest(
                worker_id=worker_id,
                labels={
                    label.key: label.to_proto()
                    for label in labels
                    if label.key is not None
                },
            ),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=create_authorization_header(self.token),
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
        await self.aio_client.UpsertWorkerLabels(  # type: ignore[misc]
            UpsertWorkerLabelsRequest(worker_id=worker_id, labels=worker_labels),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=create_authorization_header(self.token),
        )
