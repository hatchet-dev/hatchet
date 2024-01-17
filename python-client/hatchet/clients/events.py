from ..events_pb2_grpc import EventsServiceStub
from ..events_pb2 import PushEventRequest

import datetime
from ..loader import ClientConfig
import json
import grpc
from google.protobuf import timestamp_pb2

def new_event(conn, config: ClientConfig):
    return EventClientImpl(
        client=EventsServiceStub(conn),
        tenant_id=config.tenant_id,
        # logger=shared_opts['logger'],
        # validator=shared_opts['validator'],
    )

class EventClientImpl:
    def __init__(self, client, tenant_id):
        self.client = client
        self.tenant_id = tenant_id
        # self.logger = logger
        # self.validator = validator

    def push(self, event_key, payload):
        try:
            payload_bytes = json.dumps(payload).encode('utf-8')
        except json.UnicodeEncodeError as e:
            raise ValueError(f"Error encoding payload: {e}")

        request = PushEventRequest(
            tenantId=self.tenant_id,
            key=event_key,
            payload=payload_bytes,
            eventTimestamp=timestamp_pb2.Timestamp().FromDatetime(datetime.datetime.now()),
        )

        try:
            self.client.Push(request)
        except grpc.RpcError as e:
            raise ValueError(f"gRPC error: {e}")
