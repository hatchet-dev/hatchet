from datetime import datetime
from typing import List, Union
import grpc
from google.protobuf import timestamp_pb2
from ..workflows_pb2_grpc import WorkflowServiceStub
from ..workflows_pb2 import CreateWorkflowVersionOpts, ScheduleWorkflowRequest, TriggerWorkflowRequest, PutWorkflowRequest, TriggerWorkflowResponse
from ..loader import ClientConfig
from ..metadata import get_metadata
import json


def new_admin(conn, config: ClientConfig):
    return AdminClientImpl(
        client=WorkflowServiceStub(conn),
        token=config.token,
    )


class AdminClientImpl:
    def __init__(self, client: WorkflowServiceStub, token):
        self.client = client
        self.token = token

    def put_workflow(self, workflow: CreateWorkflowVersionOpts):
        try:
            self.client.PutWorkflow(
                PutWorkflowRequest(
                    opts=workflow,
                ),
                metadata=get_metadata(self.token),
            )
        except grpc.RpcError as e:
            raise ValueError(f"Could not put workflow: {e}")



    def schedule_workflow(self, name: str, schedules: List[Union[datetime, timestamp_pb2.Timestamp]], input={}):
        timestamp_schedules = []
        for schedule in schedules:
            if isinstance(schedule, datetime):
                t = schedule.timestamp()
                seconds = int(t)
                nanos = int(t % 1 * 1e9)
                timestamp = timestamp_pb2.Timestamp(seconds=seconds, nanos=nanos)
                timestamp_schedules.append(timestamp)
            elif isinstance(schedule, timestamp_pb2.Timestamp):
                timestamp_schedules.append(schedule)
            else:
                raise ValueError("Invalid schedule type. Must be datetime or timestamp_pb2.Timestamp.")

        try:
            self.client.ScheduleWorkflow(ScheduleWorkflowRequest(
                name=name,
                schedules=timestamp_schedules,
                input=json.dumps(input),
            ), metadata=get_metadata(self.token))

        except grpc.RpcError as e:
            raise ValueError(f"gRPC error: {e}")

    def run_workflow(self, workflow_name: str, input: any):
        try:
            payload_data = json.dumps(input)

            resp: TriggerWorkflowResponse = self.client.TriggerWorkflow(TriggerWorkflowRequest(
                name=workflow_name,
                input=payload_data,
            ), metadata=get_metadata(self.token))

            return resp.workflow_run_id
        except grpc.RpcError as e:
            raise ValueError(f"gRPC error: {e}")
        except json.JSONDecodeError as e:
            raise ValueError(f"Error encoding payload: {e}")
