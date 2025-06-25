import asyncio
import datetime
import json
from typing import cast

from google.protobuf import timestamp_pb2
from pydantic import BaseModel, Field

from hatchet_sdk.clients.rest.api.event_api import EventApi
from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi
from hatchet_sdk.clients.rest.api_client import ApiClient
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
    Event,
    Events,
    PushEventRequest,
    PutLogRequest,
    PutStreamEventRequest,
)
from hatchet_sdk.contracts.events_pb2_grpc import EventsServiceStub
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.utils.typing import JSONSerializableMapping


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
    @tenacity_retry
    def push(
        self,
        event_key: str,
        payload: JSONSerializableMapping,
        options: PushEventOptions = PushEventOptions(),
    ) -> Event:
        namespace = options.namespace or self.namespace
        namespaced_event_key = self.client_config.apply_namespace(event_key, namespace)

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
            eventTimestamp=proto_timestamp_now(),
            additionalMetadata=meta_bytes,
            priority=options.priority,
            scope=options.scope,
        )

        return cast(
            Event,
            self.events_service_client.Push(request, metadata=get_metadata(self.token)),
        )

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
            eventTimestamp=proto_timestamp_now(),
            additionalMetadata=meta_str,
            priority=event.priority,
            scope=event.scope,
        )

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    def bulk_push(
        self,
        events: list[BulkPushEventWithMetadata],
        options: BulkPushEventOptions = BulkPushEventOptions(),
    ) -> list[Event]:
        namespace = options.namespace or self.namespace

        bulk_request = BulkPushEventRequest(
            events=[
                self._create_push_event_request(event, namespace) for event in events
            ]
        )

        return list(
            cast(
                Events,
                self.events_service_client.BulkPush(
                    bulk_request, metadata=get_metadata(self.token)
                ),
            ).events
        )

    @tenacity_retry
    def log(self, message: str, step_run_id: str) -> None:
        request = PutLogRequest(
            stepRunId=step_run_id,
            createdAt=proto_timestamp_now(),
            message=message,
        )

        self.events_service_client.PutLog(request, metadata=get_metadata(self.token))

    @tenacity_retry
    def stream(self, data: str | bytes, step_run_id: str, index: int) -> None:
        if isinstance(data, str):
            data_bytes = data.encode("utf-8")
        elif isinstance(data, bytes):
            data_bytes = data
        else:
            raise ValueError("Invalid data type. Expected str, bytes, or file.")

        request = PutStreamEventRequest(
            stepRunId=step_run_id,
            createdAt=proto_timestamp_now(),
            message=data_bytes,
            eventIndex=index,
        )

        try:
            self.events_service_client.PutStreamEvent(
                request, metadata=get_metadata(self.token)
            )
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
