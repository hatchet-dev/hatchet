import asyncio
import datetime
import json
from typing import Any, Dict, List, Optional, TypedDict
from uuid import uuid4

import grpc
from google.protobuf import timestamp_pb2

from hatchet_sdk.contracts.events_pb2 import (
    BulkPushEventRequest,
    Event,
    PushEventRequest,
    PutLogRequest,
    PutStreamEventRequest,
)
from hatchet_sdk.contracts.events_pb2_grpc import EventsServiceStub
from hatchet_sdk.v0.clients.rest.tenacity_utils import tenacity_retry

from ..loader import ClientConfig
from ..metadata import get_metadata


def new_event(conn, config: ClientConfig):
    return EventClient(
        client=EventsServiceStub(conn),
        config=config,
    )


def proto_timestamp_now():
    t = datetime.datetime.now().timestamp()
    seconds = int(t)
    nanos = int(t % 1 * 1e9)

    return timestamp_pb2.Timestamp(seconds=seconds, nanos=nanos)


class PushEventOptions(TypedDict, total=False):
    additional_metadata: Dict[str, str] | None = None
    namespace: str | None = None


class BulkPushEventOptions(TypedDict, total=False):
    namespace: str | None = None


class BulkPushEventWithMetadata(TypedDict, total=False):
    key: str
    payload: Any
    additional_metadata: Optional[Dict[str, Any]]  # Optional metadata


class EventClient:
    def __init__(self, client: EventsServiceStub, config: ClientConfig):
        self.client = client
        self.token = config.token
        self.namespace = config.namespace

    async def async_push(
        self, event_key, payload, options: Optional[PushEventOptions] = None
    ) -> Event:
        return await asyncio.to_thread(
            self.push, event_key=event_key, payload=payload, options=options
        )

    async def async_bulk_push(
        self,
        events: List[BulkPushEventWithMetadata],
        options: Optional[BulkPushEventOptions] = None,
    ) -> List[Event]:
        return await asyncio.to_thread(self.bulk_push, events=events, options=options)

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    def push(self, event_key, payload, options: PushEventOptions = None) -> Event:
        namespace = self.namespace

        if (
            options is not None
            and "namespace" in options
            and options["namespace"] is not None
        ):
            namespace = options.pop("namespace")

        namespaced_event_key = namespace + event_key

        try:
            meta = dict() if options is None else options["additional_metadata"]
            meta_bytes = None if meta is None else json.dumps(meta).encode("utf-8")
        except Exception as e:
            raise ValueError(f"Error encoding meta: {e}")

        try:
            payload_bytes = json.dumps(payload).encode("utf-8")
        except json.UnicodeEncodeError as e:
            raise ValueError(f"Error encoding payload: {e}")

        request = PushEventRequest(
            key=namespaced_event_key,
            payload=payload_bytes,
            eventTimestamp=proto_timestamp_now(),
            additionalMetadata=meta_bytes,
        )

        return self.client.Push(request, metadata=get_metadata(self.token))

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    def bulk_push(
        self,
        events: List[BulkPushEventWithMetadata],
        options: BulkPushEventOptions = None,
    ) -> List[Event]:
        namespace = self.namespace

        if (
            options is not None
            and "namespace" in options
            and options["namespace"] is not None
        ):
            namespace = options.pop("namespace")

        bulk_events = []
        for event in events:
            event_key = namespace + event["key"]
            payload = event["payload"]

            try:
                meta = event.get("additional_metadata", {})
                meta_bytes = json.dumps(meta).encode("utf-8") if meta else None
            except Exception as e:
                raise ValueError(f"Error encoding meta: {e}")

            try:
                payload_bytes = json.dumps(payload).encode("utf-8")
            except json.UnicodeEncodeError as e:
                raise ValueError(f"Error encoding payload: {e}")

            request = PushEventRequest(
                key=event_key,
                payload=payload_bytes,
                eventTimestamp=proto_timestamp_now(),
                additionalMetadata=meta_bytes,
            )
            bulk_events.append(request)

        bulk_request = BulkPushEventRequest(events=bulk_events)

        response = self.client.BulkPush(bulk_request, metadata=get_metadata(self.token))

        return response.events

    def log(self, message: str, step_run_id: str):
        try:
            request = PutLogRequest(
                stepRunId=step_run_id,
                createdAt=proto_timestamp_now(),
                message=message,
            )

            self.client.PutLog(request, metadata=get_metadata(self.token))
        except Exception as e:
            raise ValueError(f"Error logging: {e}")

    def stream(self, data: str | bytes, step_run_id: str):
        try:
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
            )
            self.client.PutStreamEvent(request, metadata=get_metadata(self.token))
        except Exception as e:
            raise ValueError(f"Error putting stream event: {e}")
