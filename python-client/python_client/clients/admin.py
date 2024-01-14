from typing import List
import grpc
from google.protobuf import timestamp_pb2
from workflows_pb2_grpc import WorkflowServiceStub
from workflows_pb2 import CreateWorkflowVersionOpts, ScheduleWorkflowRequest, PutWorkflowRequest
from loader import ClientConfig

def new_admin(conn, config: ClientConfig):
    return AdminClientImpl(
        client=WorkflowServiceStub(conn),
        tenant_id=config.tenant_id,
        # logger=shared_opts['logger'],
        # validator=shared_opts['validator'],
    )

class AdminClientImpl:
    def __init__(self, client : WorkflowServiceStub, tenant_id):
        self.client = client
        self.tenant_id = tenant_id
        # self.logger = logger
        # self.validator = validator

    def put_workflow(self, workflow: CreateWorkflowVersionOpts):
        try:
            self.client.PutWorkflow(PutWorkflowRequest(
                tenant_id=self.tenant_id,
                opts=workflow,
            ))
        except grpc.RpcError as e:
            raise ValueError(f"gRPC error: {e}")
    
    def schedule_workflow(self, workflow_id : str, schedules : List[timestamp_pb2.Timestamp]):
        try:
            self.client.ScheduleWorkflow(ScheduleWorkflowRequest(
                tenant_id=self.tenant_id,
                workflow_id=workflow_id,
                schedules=schedules,
            ))
        except grpc.RpcError as e:
            raise ValueError(f"gRPC error: {e}")
