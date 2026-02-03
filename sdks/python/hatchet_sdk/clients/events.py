import asyncio
import datetime
import json
from typing import cast

from google.protobuf import timestamp_pb2
from pydantic import BaseModel, ConfigDict, Field

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
)
from hatchet_sdk.contracts.events_pb2 import Event as EventProto
from hatchet_sdk.contracts.events_pb2 import Events as EventsProto
from hatchet_sdk.contracts.events_pb2_grpc import EventsServiceStub
from hatchet_sdk.logger import logger
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.utils.typing import JSONSerializableMapping, LogLevel


def proto_timestamp_now() -> timestamp_pb2.Timestamp:
    t = datetime.datetime.now(tz=datetime.timezone.utc).timestamp()
    seconds = int(t)
    nanos = int(t % 1 * 1e9)

    return timestamp_pb2.Timestamp(seconds=seconds, nanos=nanos)


class PushEventOptions(BaseModel):
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)
    namespace: str | None = None
    priority: int | None = None
    scope: str | None = None


class BulkPushEventOptions(BaseModel):
    namespace: str | None = None


class BulkPushEventWithMetadata(BaseModel):
    key: str
    payload: JSONSerializableMapping = Field(default_factory=dict)
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)
    priority: int | None = None
    scope: str | None = None


class Event(BaseModel):
    tenant_id: str
    event_id: str
    key: str
    payload: str
    event_timestamp: timestamp_pb2.Timestamp
    additional_metadata: str | None = None
    scope: str | None = None

    model_config = ConfigDict(arbitrary_types_allowed=True)

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
        )


class EventClient(BaseRestClient):
    def __init__(self, config: ClientConfig):
        super().__init__(config)

        conn = new_conn(config, False)
        self.events_service_client = EventsServiceStub(conn)

        self.token = config.token
        self.namespace = config.namespace

    def _wra(self, client: ApiClient) -> WorkflowRunsApi:
        return WorkflowRunsApi(client)

    def _ea(self, client: ApiClient) -> EventApi:
        return EventApi(client)

    async def aio_push(
        self,
        event_key: str,
        payload: JSONSerializableMapping,
        options: PushEventOptions = PushEventOptions(),
    ) -> Event:
        return await asyncio.to_thread(
            self.push, event_key=event_key, payload=payload, options=options
        )

    async def aio_bulk_push(
        self,
        events: list[BulkPushEventWithMetadata],
        options: BulkPushEventOptions = BulkPushEventOptions(),
    ) -> list[Event]:
        return await asyncio.to_thread(self.bulk_push, events=events, options=options)

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    def push(
        self,
        event_key: str,
        payload: JSONSerializableMapping,
        options: PushEventOptions = PushEventOptions(),
    ) -> Event:
        namespace = options.namespace or self.namespace
        namespaced_event_key = self.client_config.apply_namespace(event_key, namespace)
        push_event = tenacity_retry(
            self.events_service_client.Push, self.client_config.tenacity
        )

        try:
            meta_bytes = json.dumps(options.additional_metadata)
        except Exception as e:
            raise ValueError("Error encoding meta") from e

        try:
            payload_str = json.dumps(payload)
        except (TypeError, ValueError) as e:
            raise ValueError("Error encoding payload") from e

        request = PushEventRequest(
            key=namespaced_event_key,
            payload=payload_str,
            event_timestamp=proto_timestamp_now(),
            additional_metadata=meta_bytes,
            priority=options.priority,
            scope=options.scope,
        )

        response = cast(
            EventProto,
            push_event(request, metadata=get_metadata(self.token)),
        )
        return Event.from_proto(response)

    def _create_push_event_request(
        self,
        event: BulkPushEventWithMetadata,
        namespace: str,
    ) -> PushEventRequest:
        event_key = self.client_config.apply_namespace(event.key, namespace)
        payload = event.payload

        meta = event.additional_metadata

        try:
            meta_str = json.dumps(meta)
        except Exception as e:
            raise ValueError("Error encoding meta") from e

        try:
            serialized_payload = json.dumps(payload)
        except (TypeError, ValueError) as e:
            raise ValueError("Error serializing payload") from e

        return PushEventRequest(
            key=event_key,
            payload=serialized_payload,
            event_timestamp=proto_timestamp_now(),
            additional_metadata=meta_str,
            priority=event.priority,
            scope=event.scope,
        )

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    def bulk_push(
        self,
        events: list[BulkPushEventWithMetadata],
        options: BulkPushEventOptions = BulkPushEventOptions(),
    ) -> list[Event]:
        namespace = options.namespace or self.namespace
        bulk_push = tenacity_retry(
            self.events_service_client.BulkPush, self.client_config.tenacity
        )

        bulk_request = BulkPushEventRequest(
            events=[
                self._create_push_event_request(event, namespace) for event in events
            ]
        )

        response = cast(
            EventsProto,
            bulk_push(bulk_request, metadata=get_metadata(self.token)),
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

        put_log = tenacity_retry(
            self.events_service_client.PutLog, self.client_config.tenacity
        )
        request = PutLogRequest(
            task_run_id=step_run_id,
            created_at=proto_timestamp_now(),
            message=message,
            level=level.value if level else None,
            task_retry_count=task_retry_count,
        )

        put_log(request, metadata=get_metadata(self.token))

    def stream(self, data: str | bytes, step_run_id: str, index: int) -> None:
        put_stream_event = tenacity_retry(
            self.events_service_client.PutStreamEvent, self.client_config.tenacity
        )
        if isinstance(data, str):
            data_bytes = data.encode("utf-8")
        elif isinstance(data, bytes):
            data_bytes = data
        else:
            raise ValueError("Invalid data type. Expected str, bytes, or file.")

        request = PutStreamEventRequest(
            task_run_id=step_run_id,
            created_at=proto_timestamp_now(),
            message=data_bytes,
            event_index=index,
        )

        try:
            put_stream_event(request, metadata=get_metadata(self.token))
        except Exception:
            raise

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
