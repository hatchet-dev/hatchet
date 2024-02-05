from typing import List
import grpc
from google.protobuf import timestamp_pb2
from ..workflows_pb2_grpc import WorkflowServiceStub
from ..workflows_pb2 import CreateWorkflowVersionOpts, ScheduleWorkflowRequest, TriggerWorkflowRequest, PutWorkflowRequest, TriggerWorkflowResponse
from ..loader import ClientConfig
from ..semver import bump_minor_version
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

    def schedule_workflow(self, workflow_id: str, schedules: List[timestamp_pb2.Timestamp]):
        try:
            self.client.ScheduleWorkflow(ScheduleWorkflowRequest(
                workflow_id=workflow_id,
                schedules=schedules,
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
