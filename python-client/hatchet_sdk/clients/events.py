from ..events_pb2_grpc import EventsServiceStub
from ..events_pb2 import PushEventRequest

import datetime
from ..loader import ClientConfig
import json
import grpc
from google.protobuf import timestamp_pb2
from ..metadata import get_metadata

def new_event(conn, config: ClientConfig):
    return EventClientImpl(
        client=EventsServiceStub(conn),
        token=config.token,
    )

class EventClientImpl:
    def __init__(self, client, token):
        self.client = client
        self.token = token

    def push(self, event_key, payload):
        try:
            payload_bytes = json.dumps(payload).encode('utf-8')
        except json.UnicodeEncodeError as e:
            raise ValueError(f"Error encoding payload: {e}")

        request = PushEventRequest(
            key=event_key,
            payload=payload_bytes,
            eventTimestamp=timestamp_pb2.Timestamp().FromDatetime(datetime.datetime.now()),
        )

        try:
            self.client.Push(request, metadata=get_metadata(self.token))
        except grpc.RpcError as e:
            raise ValueError(f"gRPC error: {e}")
