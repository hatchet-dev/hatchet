from typing import List
import grpc
from google.protobuf import timestamp_pb2
from ..workflows_pb2_grpc import WorkflowServiceStub
from ..workflows_pb2 import CreateWorkflowVersionOpts, ScheduleWorkflowRequest, PutWorkflowRequest, GetWorkflowByNameRequest, Workflow
from ..loader import ClientConfig
from ..semver import bump_minor_version
from ..metadata import get_metadata


def new_admin(conn, config: ClientConfig):
    return AdminClientImpl(
        client=WorkflowServiceStub(conn),
        token=config.token,
    )

class AdminClientImpl:
    def __init__(self, client : WorkflowServiceStub, token):
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
    
    def schedule_workflow(self, workflow_id : str, schedules : List[timestamp_pb2.Timestamp]):
        try:
            self.client.ScheduleWorkflow(ScheduleWorkflowRequest(
                tenant_id=self.tenant_id,
                workflow_id=workflow_id,
                schedules=schedules,
            ), metadata=get_metadata(self.token))
        except grpc.RpcError as e:
            raise ValueError(f"gRPC error: {e}")
