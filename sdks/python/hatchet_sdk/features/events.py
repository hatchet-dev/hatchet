import asyncio
import datetime
import json
from typing import TYPE_CHECKING, cast

from pydantic import BaseModel, Field

if TYPE_CHECKING:
    from hatchet_sdk.clients.rest.api.event_api import EventApi
    from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi

from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_event import V1Event
from hatchet_sdk.clients.rest.models.v1_event_triggered_run import V1EventTriggeredRun
from hatchet_sdk.clients.rest.models.v1_event_workflow_run_summary import (
    V1EventWorkflowRunSummary,
)
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import (
    BaseRestClient,
    maybe_additional_metadata_to_kv,
)
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.events_pb2 import (
    BulkPushEventRequest,
    PushEventRequest,
    PutLogRequest,
    PutStreamEventRequest,
    PutStreamEventResponse,
)
from hatchet_sdk.contracts.events_pb2 import Event as EventProto
from hatchet_sdk.contracts.events_pb2 import Events as EventsProto
from hatchet_sdk.contracts.events_pb2_grpc import EventsServiceStub
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.contextvars import (
    ctx_task_run_external_id,
    ctx_workflow_run_id,
)
from hatchet_sdk.types.priority import Priority
from hatchet_sdk.types.run_status import RunStatus
from hatchet_sdk.types.trigger import (
    BulkPushEventWithMetadata as BulkPushEventWithMetadata,
)
from hatchet_sdk.utils.api_auth import create_authorization_header
from hatchet_sdk.utils.datetimes import proto_timestamp_now
from hatchet_sdk.utils.typing import JSONSerializableMapping, LogLevel


def _inject_source_info(
    metadata: JSONSerializableMapping,
) -> JSONSerializableMapping:
    """Injects hatchet__source_workflow_run_id and hatchet__source_step_run_id
    into metadata when called from within a task execution context. The engine reads
    these exact keys, so the `step` in the second one cannot be renamed here."""
    wf_run_id = ctx_workflow_run_id.get()
    task_run_external_id = ctx_task_run_external_id.get()

    if not wf_run_id or not task_run_external_id:
        return metadata

    return {
        **metadata,
        "hatchet__source_workflow_run_id": wf_run_id,
        "hatchet__source_step_run_id": task_run_external_id,
    }


class Event(BaseModel):
    event_id: str
    tenant_id: str
    key: str
    payload: JSONSerializableMapping
    additional_metadata: JSONSerializableMapping
    scope: str | None = None
    seen_at: datetime.datetime
    triggering_webhook_name: str | None = None
    workflow_run_summary: V1EventWorkflowRunSummary | None = Field(
        default=None,
        description="Counts of the runs this event triggered, by status. Only set on events read back from Hatchet, not on the result of a push.",
    )
    triggered_runs: list[V1EventTriggeredRun] | None = Field(
        default=None,
        description="The runs this event triggered. Only set on events read back from Hatchet, not on the result of a push.",
    )
    payloads_restricted: bool = Field(
        default=False,
        description="True when the payload was omitted because the caller cannot view payloads.",
    )

    @classmethod
    def from_proto(cls, proto: EventProto) -> "Event":
        return cls(
            event_id=proto.event_id,
            tenant_id=proto.tenant_id,
            key=proto.key,
            payload=json.loads(proto.payload) if proto.payload else {},
            additional_metadata=(
                json.loads(proto.additional_metadata)
                if proto.HasField("additional_metadata") and proto.additional_metadata
                else {}
            ),
            scope=proto.scope if proto.HasField("scope") else None,
            seen_at=proto.event_timestamp.ToDatetime(tzinfo=datetime.UTC),
        )

    @classmethod
    def from_rest_model(cls, rest_event: V1Event) -> "Event":
        return cls(
            event_id=rest_event.metadata.id,
            tenant_id=rest_event.tenant_id,
            key=rest_event.key,
            payload=rest_event.payload or {},
            additional_metadata=rest_event.additional_metadata or {},
            scope=rest_event.scope,
            seen_at=rest_event.seen_at or rest_event.metadata.created_at,
            triggering_webhook_name=rest_event.triggering_webhook_name,
            workflow_run_summary=rest_event.workflow_run_summary,
            triggered_runs=rest_event.triggered_runs,
            payloads_restricted=rest_event.payloads_restricted is True,
        )


class EventClient(BaseRestClient):
    """
    The events client is a client for pushing events to Hatchet and reading the events that were pushed.
    """

    def __init__(self, config: ClientConfig) -> None:
        super().__init__(config)

        self._client: EventsServiceStub | None = None
        self._aio_client: EventsServiceStub | None = None

        self._token = config.token
        self._namespace = config.namespace
        self._retrying_aio_put_stream_event = tenacity_retry(
            self._put_stream_event, self.client_config.tenacity
        )
        self._retrying_aio_push_event = tenacity_retry(
            self._push_event, self.client_config.tenacity
        )
        self._retrying_aio_bulk_push_event = tenacity_retry(
            self._bulk_push_event, self.client_config.tenacity
        )

    def _wra(self, client: ApiClient) -> "WorkflowRunsApi":
        from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi

        return WorkflowRunsApi(client)

    def _ea(self, client: ApiClient) -> "EventApi":
        from hatchet_sdk.clients.rest.api.event_api import EventApi

        return EventApi(client)

    def _get_or_create_aio_client(self) -> EventsServiceStub:
        if self._aio_client is None:
            self._aio_client = EventsServiceStub(new_conn(self.client_config, True))

        return self._aio_client

    def _get_or_create_client(self) -> EventsServiceStub:
        if self._client is None:
            self._client = EventsServiceStub(new_conn(self.client_config, False))

        return self._client

    def _prepare_push_event_request(
        self,
        key: str,
        payload: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: Priority | None = None,
        scope: str | None = None,
    ) -> PushEventRequest:
        namespace = self._namespace
        namespaced_key = self.client_config.apply_namespace(key, namespace)

        try:
            meta = _inject_source_info(additional_metadata or {})
            meta_bytes = json.dumps(meta)
        except Exception as e:
            raise ValueError("Error encoding meta") from e

        try:
            payload_str = json.dumps(payload)
        except (TypeError, ValueError) as e:
            raise ValueError("Error encoding payload") from e

        return PushEventRequest(
            key=namespaced_key,
            payload=payload_str,
            event_timestamp=proto_timestamp_now(),
            additional_metadata=meta_bytes,
            priority=priority,
            scope=scope,
        )

    async def aio_push(
        self,
        event_key: str,
        payload: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: Priority | None = None,
        scope: str | None = None,
    ) -> Event:
        """
        Push an event to Hatchet, triggering any workflows that listen for its key.

        :param event_key: The key of the event. Workflows declared with this key in `on_events` are triggered.
        :param payload: The payload of the event, which becomes the input of the triggered runs.
        :param additional_metadata: Additional metadata to attach to the event and to the runs it triggers.
        :param priority: The priority to give the runs triggered by the event.
        :param scope: The scope of the event, used to match it against event filters.
        :return: The event that was pushed.
        """
        request = self._prepare_push_event_request(
            key=event_key,
            payload=payload,
            additional_metadata=additional_metadata,
            priority=priority,
            scope=scope,
        )

        response = await self._retrying_aio_push_event(
            request, create_authorization_header(self._token)
        )

        return Event.from_proto(response)

    async def _push_event(
        self,
        request: PushEventRequest,
        metadata: tuple[tuple[str, str]],
    ) -> EventProto:
        aio_client = self._get_or_create_aio_client()
        return cast(
            "EventProto",
            await aio_client.Push(request, metadata=metadata),  # type: ignore[misc]
        )

    def push(
        self,
        event_key: str,
        payload: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: Priority | None = None,
        scope: str | None = None,
    ) -> Event:
        """
        Push an event to Hatchet, triggering any workflows that listen for its key.

        :param event_key: The key of the event. Workflows declared with this key in `on_events` are triggered.
        :param payload: The payload of the event, which becomes the input of the triggered runs.
        :param additional_metadata: Additional metadata to attach to the event and to the runs it triggers.
        :param priority: The priority to give the runs triggered by the event.
        :param scope: The scope of the event, used to match it against event filters.
        :return: The event that was pushed.
        """

        client = self._get_or_create_client()
        push_event = tenacity_retry(client.Push, self.client_config.tenacity)

        request = self._prepare_push_event_request(
            key=event_key,
            payload=payload,
            additional_metadata=additional_metadata,
            priority=priority,
            scope=scope,
        )

        response = cast(
            "EventProto",
            push_event(request, metadata=create_authorization_header(self._token)),
        )

        return Event.from_proto(response)

    async def aio_bulk_push(
        self,
        events: list[BulkPushEventWithMetadata],
    ) -> list[Event]:
        """
        Push many events to Hatchet in a single request.

        :param events: The events to push.
        :return: The events that were pushed.
        """
        bulk_request = BulkPushEventRequest(
            events=[
                self._prepare_push_event_request(
                    key=event.key,
                    payload=event.payload,
                    additional_metadata=event.additional_metadata,
                    priority=(
                        Priority(event.priority)
                        if isinstance(event.priority, int)
                        else event.priority
                    ),
                    scope=event.scope,
                )
                for event in events
            ]
        )

        response = await self._retrying_aio_bulk_push_event(
            bulk_request, create_authorization_header(self._token)
        )

        return [Event.from_proto(event) for event in response.events]

    async def _bulk_push_event(
        self,
        request: BulkPushEventRequest,
        metadata: tuple[tuple[str, str]],
    ) -> EventsProto:
        aio_client = self._get_or_create_aio_client()
        return cast(
            "EventsProto",
            await aio_client.BulkPush(request, metadata=metadata),  # type: ignore[misc]
        )

    def bulk_push(
        self,
        events: list[BulkPushEventWithMetadata],
    ) -> list[Event]:
        """
        Push many events to Hatchet in a single request.

        :param events: The events to push.
        :return: The events that were pushed.
        """
        bulk_request = BulkPushEventRequest(
            events=[
                self._prepare_push_event_request(
                    key=event.key,
                    payload=event.payload,
                    additional_metadata=event.additional_metadata,
                    priority=(
                        Priority(event.priority)
                        if isinstance(event.priority, int)
                        else event.priority
                    ),
                    scope=event.scope,
                )
                for event in events
            ]
        )

        client = self._get_or_create_client()

        bulk_push = tenacity_retry(client.BulkPush, self.client_config.tenacity)

        response = cast(
            "EventsProto",
            bulk_push(bulk_request, metadata=create_authorization_header(self._token)),
        )

        return [Event.from_proto(event) for event in response.events]

    def _log(
        self,
        message: str,
        task_run_external_id: str,
        level: LogLevel | None = None,
        task_retry_count: int | None = None,
    ) -> None:
        if len(message) > 10_000:
            logger.warning("truncating log message to 10,000 characters")
            message = message[:10_000]

        client = self._get_or_create_client()
        put_log = tenacity_retry(client.PutLog, self.client_config.tenacity)
        request = PutLogRequest(
            task_run_external_id=task_run_external_id,
            created_at=proto_timestamp_now(),
            message=message,
            level=level.value if level else None,
            task_retry_count=task_retry_count,
        )

        put_log(request, metadata=create_authorization_header(self._token))

    def _create_put_stream_event_request(
        self, data: str | bytes, task_run_external_id: str, index: int
    ) -> PutStreamEventRequest:
        if isinstance(data, str):
            data_bytes = data.encode("utf-8")
        elif isinstance(data, bytes):
            data_bytes = data
        else:
            raise ValueError("Invalid data type. Expected str, bytes, or file.")

        return PutStreamEventRequest(
            task_run_external_id=task_run_external_id,
            created_at=proto_timestamp_now(),
            message=data_bytes,
            event_index=index,
        )

    def _stream(self, data: str | bytes, task_run_external_id: str, index: int) -> None:
        client = self._get_or_create_client()
        put_stream_event = tenacity_retry(
            client.PutStreamEvent, self.client_config.tenacity
        )
        request = self._create_put_stream_event_request(
            data, task_run_external_id, index
        )

        try:
            put_stream_event(request, metadata=create_authorization_header(self._token))
        except Exception:
            raise

    async def _put_stream_event(
        self,
        request: PutStreamEventRequest,
        metadata: tuple[tuple[str, str]],
    ) -> PutStreamEventResponse:
        client = self._get_or_create_aio_client()
        return cast(
            "PutStreamEventResponse",
            await client.PutStreamEvent(  # type: ignore[misc]
                request, metadata=metadata
            ),
        )

    async def _aio_stream(
        self, data: str | bytes, task_run_external_id: str, index: int
    ) -> None:
        request = self._create_put_stream_event_request(
            data, task_run_external_id, index
        )

        await self._retrying_aio_put_stream_event(
            request, create_authorization_header(self._token)
        )

    async def aio_list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        keys: list[str] | None = None,
        since: datetime.datetime | None = None,
        until: datetime.datetime | None = None,
        workflow_ids: list[str] | None = None,
        workflow_run_statuses: list[RunStatus] | None = None,
        event_ids: list[str] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        scopes: list[str] | None = None,
    ) -> list[Event]:
        """
        List events that were pushed to the tenant.

        :param offset: The number of events to skip.
        :param limit: The maximum number of events to return.
        :param keys: Only return events with one of these keys.
        :param since: Only return events seen at or after this time.
        :param until: Only return events seen at or before this time.
        :param workflow_ids: Only return events that triggered one of these workflows.
        :param workflow_run_statuses: Only return events that triggered a run in one of these statuses.
        :param event_ids: Only return events with one of these IDs.
        :param additional_metadata: Only return events whose additional metadata contains these key-value pairs.
        :param scopes: Only return events with one of these scopes.
        :return: The matching events.
        """
        return await asyncio.to_thread(
            self.list,
            offset=offset,
            limit=limit,
            keys=keys,
            since=since,
            until=until,
            workflow_ids=workflow_ids,
            workflow_run_statuses=workflow_run_statuses,
            event_ids=event_ids,
            additional_metadata=additional_metadata,
            scopes=scopes,
        )

    def list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        keys: list[str] | None = None,
        since: datetime.datetime | None = None,
        until: datetime.datetime | None = None,
        workflow_ids: list[str] | None = None,
        workflow_run_statuses: list[RunStatus] | None = None,
        event_ids: list[str] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        scopes: list[str] | None = None,
    ) -> list[Event]:
        """
        List events that were pushed to the tenant.

        :param offset: The number of events to skip.
        :param limit: The maximum number of events to return.
        :param keys: Only return events with one of these keys.
        :param since: Only return events seen at or after this time.
        :param until: Only return events seen at or before this time.
        :param workflow_ids: Only return events that triggered one of these workflows.
        :param workflow_run_statuses: Only return events that triggered a run in one of these statuses.
        :param event_ids: Only return events with one of these IDs.
        :param additional_metadata: Only return events whose additional metadata contains these key-value pairs.
        :param scopes: Only return events with one of these scopes.
        :return: The matching events.
        """
        with self.client() as client:
            el = self._ea(client).v1_event_list(
                tenant=self.client_config.tenant_id,
                offset=offset,
                limit=limit,
                keys=keys,
                since=since,
                until=until,
                workflow_ids=workflow_ids,
                workflow_run_statuses=workflow_run_statuses,
                event_ids=event_ids,
                additional_metadata=maybe_additional_metadata_to_kv(
                    additional_metadata
                ),
                scopes=scopes,
            )

            return [Event.from_rest_model(rest_event) for rest_event in el.rows or []]

    def get(
        self,
        event_id: str,
    ) -> Event:
        """
        Retrieve an event by its ID.

        :param event_id: The ID of the event to retrieve.
        :return: The requested event.
        """
        with self.client() as client:
            return Event.from_rest_model(
                self._ea(client).v1_event_get(
                    tenant=self.client_config.tenant_id,
                    v1_event=event_id,
                )
            )

    async def aio_get(
        self,
        event_id: str,
    ) -> Event:
        """
        Retrieve an event by its ID.

        :param event_id: The ID of the event to retrieve.
        :return: The requested event.
        """
        return await asyncio.to_thread(
            self.get,
            event_id=event_id,
        )
