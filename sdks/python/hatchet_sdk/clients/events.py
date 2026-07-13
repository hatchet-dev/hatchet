import asyncio
import datetime
import json
import warnings
from datetime import timezone
from typing import cast

from google.protobuf import timestamp_pb2
from pydantic import BaseModel, ConfigDict

from hatchet_sdk.clients.rest.api.event_api import EventApi
from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_event import V1Event
from hatchet_sdk.clients.rest.models.v1_event_list import V1EventList
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
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
from hatchet_sdk.runnables.contextvars import ctx_step_run_id, ctx_workflow_run_id
from hatchet_sdk.types.priority import Priority
from hatchet_sdk.types.trigger import (
    BulkPushEventOptions as BulkPushEventOptions,
)
from hatchet_sdk.types.trigger import (
    BulkPushEventWithMetadata as BulkPushEventWithMetadata,
)
from hatchet_sdk.types.trigger import (
    PushEventOptions as PushEventOptions,
)
from hatchet_sdk.utils.api_auth import create_authorization_header
from hatchet_sdk.utils.typing import JSONSerializableMapping, LogLevel


def _inject_source_info(
    metadata: JSONSerializableMapping,
) -> JSONSerializableMapping:
    """Injects hatchet__source_workflow_run_id and hatchet__source_step_run_id
    into metadata when called from within a step execution context."""
    wf_run_id = ctx_workflow_run_id.get()
    step_run_id = ctx_step_run_id.get()

    if not wf_run_id or not step_run_id:
        return metadata

    return {
        **metadata,
        "hatchet__source_workflow_run_id": wf_run_id,
        "hatchet__source_step_run_id": step_run_id,
    }


def proto_timestamp_now() -> timestamp_pb2.Timestamp:
    t = datetime.datetime.now(tz=datetime.timezone.utc).timestamp()
    seconds = int(t)
    nanos = int(t % 1 * 1e9)

    return timestamp_pb2.Timestamp(seconds=seconds, nanos=nanos)


class Event(BaseModel):
    tenant_id: str
    event_id: str
    key: str
    payload: str
    event_timestamp: timestamp_pb2.Timestamp
    additional_metadata: str | None = None
    scope: str | None = None
    seen_at: datetime.datetime

    model_config = ConfigDict(arbitrary_types_allowed=True)

    @property
    def eventTimestamp(self) -> timestamp_pb2.Timestamp:  # noqa: N802
        return self.event_timestamp

    @property
    def additionalMetadata(self) -> str | None:  # noqa: N802
        return self.additional_metadata

    @classmethod
    def from_proto(cls, proto: EventProto) -> "Event":
        additional_metadata = (
            proto.additional_metadata if proto.HasField("additional_metadata") else None
        )
        scope = proto.scope if proto.HasField("scope") else None

        return cls(
            tenant_id=proto.tenant_id,
            event_id=proto.event_id,
            key=proto.key,
            payload=proto.payload,
            event_timestamp=proto.event_timestamp,
            additional_metadata=additional_metadata,
            scope=scope,
            seen_at=proto.event_timestamp.ToDatetime(tzinfo=timezone.utc),
        )


class EventClient(BaseRestClient):
    def __init__(self, config: ClientConfig):
        super().__init__(config)

        self._client: EventsServiceStub | None = None
        self._aio_client: EventsServiceStub | None = None

        self.token = config.token
        self.namespace = config.namespace
        self._retrying_aio_put_stream_event = tenacity_retry(
            self._put_stream_event, self.client_config.tenacity
        )

    def _wra(self, client: ApiClient) -> WorkflowRunsApi:
        return WorkflowRunsApi(client)

    def _ea(self, client: ApiClient) -> EventApi:
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
        options: PushEventOptions,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: Priority | None = None,
        scope: str | None = None,
        namespace_override: str | None = None,
    ) -> PushEventRequest:
        namespace = namespace_override or options.namespace or self.namespace
        namespaced_key = self.client_config.apply_namespace(key, namespace)

        try:
            meta = _inject_source_info(
                additional_metadata or options.additional_metadata
            )
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
            priority=priority or options.priority,
            scope=scope or options.scope,
        )

    async def aio_push(
        self,
        event_key: str,
        payload: JSONSerializableMapping,
        options: PushEventOptions | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: Priority | None = None,
        scope: str | None = None,
    ) -> Event:
        if options is not None:
            warnings.warn(
                "The `options` parameter is deprecated and will be removed in v2.0.0. The namespace should be set on the `ClientConfig`",
                stacklevel=2,
                category=DeprecationWarning,
            )
        else:
            options = PushEventOptions()

        aio_client = self._get_or_create_aio_client()
        push_event = tenacity_retry(aio_client.Push, self.client_config.tenacity)

        request = self._prepare_push_event_request(
            key=event_key,
            payload=payload,
            options=options,
            additional_metadata=additional_metadata,
            priority=priority,
            scope=scope,
        )

        response = cast(
            EventProto,
            await push_event(request, metadata=create_authorization_header(self.token)),  # type: ignore[misc]
        )

        return Event.from_proto(response)

    def push(
        self,
        event_key: str,
        payload: JSONSerializableMapping,
        options: PushEventOptions | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: Priority | None = None,
        scope: str | None = None,
    ) -> Event:
        if options is not None:
            warnings.warn(
                "The `options` parameter is deprecated and will be removed in v2.0.0. The namespace should be set on the `ClientConfig`",
                stacklevel=2,
                category=DeprecationWarning,
            )
        else:
            options = PushEventOptions()

        client = self._get_or_create_client()
        push_event = tenacity_retry(client.Push, self.client_config.tenacity)

        request = self._prepare_push_event_request(
            key=event_key,
            payload=payload,
            options=options,
            additional_metadata=additional_metadata,
            priority=priority,
            scope=scope,
        )

        response = cast(
            EventProto,
            push_event(request, metadata=create_authorization_header(self.token)),
        )

        return Event.from_proto(response)

    async def aio_bulk_push(
        self,
        events: list[BulkPushEventWithMetadata],
        options: BulkPushEventOptions | None = None,
    ) -> list[Event]:
        if options:
            warnings.warn(
                "The `options` parameter is deprecated and will be removed in v2.0.0. The namespace should be set on the `ClientConfig`",
                stacklevel=2,
                category=DeprecationWarning,
            )
        else:
            options = BulkPushEventOptions()

        namespace = options.namespace or self.namespace

        bulk_request = BulkPushEventRequest(
            events=[
                self._prepare_push_event_request(
                    key=event.key,
                    payload=event.payload,
                    additional_metadata=event.additional_metadata,
                    options=PushEventOptions(),
                    priority=(
                        Priority(event.priority)
                        if isinstance(event.priority, int)
                        else event.priority
                    ),
                    scope=event.scope,
                    namespace_override=namespace,
                )
                for event in events
            ]
        )

        client = self._get_or_create_aio_client()

        bulk_push = tenacity_retry(client.BulkPush, self.client_config.tenacity)

        response = cast(
            EventsProto,
            await bulk_push(  # type: ignore[misc]
                bulk_request,
                metadata=create_authorization_header(self.token),
            ),
        )

        return [Event.from_proto(event) for event in response.events]

    def bulk_push(
        self,
        events: list[BulkPushEventWithMetadata],
        options: BulkPushEventOptions | None = None,
    ) -> list[Event]:
        if options:
            warnings.warn(
                "The `options` parameter is deprecated and will be removed in v2.0.0. The namespace should be set on the `ClientConfig`",
                stacklevel=2,
                category=DeprecationWarning,
            )
        else:
            options = BulkPushEventOptions()

        namespace = options.namespace or self.namespace

        bulk_request = BulkPushEventRequest(
            events=[
                self._prepare_push_event_request(
                    key=event.key,
                    payload=event.payload,
                    additional_metadata=event.additional_metadata,
                    options=PushEventOptions(),
                    priority=(
                        Priority(event.priority)
                        if isinstance(event.priority, int)
                        else event.priority
                    ),
                    scope=event.scope,
                    namespace_override=namespace,
                )
                for event in events
            ]
        )

        client = self._get_or_create_client()

        bulk_push = tenacity_retry(client.BulkPush, self.client_config.tenacity)

        response = cast(
            EventsProto,
            bulk_push(bulk_request, metadata=create_authorization_header(self.token)),
        )

        return [Event.from_proto(event) for event in response.events]

    def log(
        self,
        message: str,
        step_run_id: str,
        level: LogLevel | None = None,
        task_retry_count: int | None = None,
    ) -> None:
        if len(message) > 10_000:
            logger.warning("truncating log message to 10,000 characters")
            message = message[:10_000]

        client = self._get_or_create_client()
        put_log = tenacity_retry(client.PutLog, self.client_config.tenacity)
        request = PutLogRequest(
            task_run_external_id=step_run_id,
            created_at=proto_timestamp_now(),
            message=message,
            level=level.value if level else None,
            task_retry_count=task_retry_count,
        )

        put_log(request, metadata=create_authorization_header(self.token))

    def _create_put_stream_event_request(
        self, data: str | bytes, step_run_id: str, index: int
    ) -> PutStreamEventRequest:
        if isinstance(data, str):
            data_bytes = data.encode("utf-8")
        elif isinstance(data, bytes):
            data_bytes = data
        else:
            raise ValueError("Invalid data type. Expected str, bytes, or file.")

        return PutStreamEventRequest(
            task_run_external_id=step_run_id,
            created_at=proto_timestamp_now(),
            message=data_bytes,
            event_index=index,
        )

    def stream(self, data: str | bytes, step_run_id: str, index: int) -> None:
        client = self._get_or_create_client()
        put_stream_event = tenacity_retry(
            client.PutStreamEvent, self.client_config.tenacity
        )
        request = self._create_put_stream_event_request(data, step_run_id, index)

        try:
            put_stream_event(request, metadata=create_authorization_header(self.token))
        except Exception:
            raise

    async def _put_stream_event(
        self,
        request: PutStreamEventRequest,
        metadata: tuple[tuple[str, str]],
    ) -> PutStreamEventResponse:
        client = self._get_or_create_aio_client()
        return cast(
            PutStreamEventResponse,
            await client.PutStreamEvent(  # type: ignore[misc]
                request, metadata=metadata
            ),
        )

    async def aio_stream(self, data: str | bytes, step_run_id: str, index: int) -> None:
        request = self._create_put_stream_event_request(data, step_run_id, index)

        await self._retrying_aio_put_stream_event(
            request, create_authorization_header(self.token)
        )

    async def aio_list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        keys: list[str] | None = None,
        since: datetime.datetime | None = None,
        until: datetime.datetime | None = None,
        workflow_ids: list[str] | None = None,
        workflow_run_statuses: list[V1TaskStatus] | None = None,
        event_ids: list[str] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        scopes: list[str] | None = None,
    ) -> V1EventList:
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
        workflow_run_statuses: list[V1TaskStatus] | None = None,
        event_ids: list[str] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        scopes: list[str] | None = None,
    ) -> V1EventList:
        with self.client() as client:
            return self._ea(client).v1_event_list(
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

    def get(
        self,
        event_id: str,
    ) -> V1Event:
        with self.client() as client:
            return self._ea(client).v1_event_get(
                tenant=self.client_config.tenant_id,
                v1_event=event_id,
            )

    async def aio_get(
        self,
        event_id: str,
    ) -> V1Event:
        return await asyncio.to_thread(
            self.get,
            event_id=event_id,
        )
