from typing import List
import grpc
from google.protobuf import timestamp_pb2
from ..workflows_pb2_grpc import WorkflowServiceStub
from ..workflows_pb2 import CreateWorkflowVersionOpts, ScheduleWorkflowRequest, PutWorkflowRequest, GetWorkflowByNameRequest, Workflow
from ..loader import ClientConfig
from ..semver import bump_minor_version

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
        
    def put_workflow(self, workflow: CreateWorkflowVersionOpts, auto_version: bool = False):
        if workflow.version == "" and not auto_version:
            raise ValueError("PutWorkflow error: workflow version is required, or use with_auto_version")
        
        existing_workflow : Workflow = None

        # Get existing workflow by name
        try:
            existing_workflow : Workflow = self.client.GetWorkflowByName(
                GetWorkflowByNameRequest(
                    tenant_id=self.tenant_id,
                    name=workflow.name,
                )
            )
        except grpc.RpcError as e:
            if e.code() == grpc.StatusCode.NOT_FOUND:
                should_put = True
            else:
                raise ValueError(f"Could not get workflow: {e}")

        # Determine if we should put the workflow
        should_put = False
        if auto_version:
            if workflow.version == "":
                workflow.version = "v0.1.0"
                should_put = True
            elif existing_workflow and existing_workflow.versions:
                if auto_version:
                    workflow.version = bump_minor_version(existing_workflow.versions[0].version)
                    should_put = True
                elif existing_workflow.versions[0].version != workflow.version:
                    should_put = True
                else:
                    should_put = False
            else:
                should_put = True

        # Put the workflow if conditions are met
        if should_put:
            try:
                self.client.PutWorkflow(
                    PutWorkflowRequest(
                        tenant_id=self.tenant_id,
                        opts=workflow,
                    )
                )
            except grpc.RpcError as e:
                raise ValueError(f"Could not create/update workflow: {e}")
    
    def schedule_workflow(self, workflow_id : str, schedules : List[timestamp_pb2.Timestamp]):
        try:
            self.client.ScheduleWorkflow(ScheduleWorkflowRequest(
                tenant_id=self.tenant_id,
                workflow_id=workflow_id,
                schedules=schedules,
            ))
        except grpc.RpcError as e:
            raise ValueError(f"gRPC error: {e}")
