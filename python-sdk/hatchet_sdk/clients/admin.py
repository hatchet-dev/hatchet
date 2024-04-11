from datetime import datetime
from typing import List, Union
import grpc
from google.protobuf import timestamp_pb2
from ..workflows_pb2_grpc import WorkflowServiceStub
from ..workflows_pb2 import CreateWorkflowVersionOpts, PutRateLimitRequest, RateLimitDuration, ScheduleWorkflowRequest, TriggerWorkflowRequest, PutWorkflowRequest, TriggerWorkflowResponse, WorkflowVersion
from ..loader import ClientConfig
from ..metadata import get_metadata
import json
from typing import TypedDict, Optional
from ..workflow import WorkflowMeta

def new_admin(conn, config: ClientConfig):
    return AdminClientImpl(
        client=WorkflowServiceStub(conn),
        token=config.token,
    )

class TriggerWorkflowParentOptions(TypedDict):
    parent_id: Optional[str]
    parent_step_run_id: Optional[str]
    child_index: Optional[int]
    child_key: Optional[str]

class AdminClientImpl:
    def __init__(self, client: WorkflowServiceStub, token):
        self.client = client
        self.token = token

    def put_workflow(self, name: str, workflow: CreateWorkflowVersionOpts | WorkflowMeta, overrides: CreateWorkflowVersionOpts | None = None) -> WorkflowVersion:
        try:
            opts : CreateWorkflowVersionOpts

            if isinstance(workflow, CreateWorkflowVersionOpts):
                opts = workflow
            else:
                opts = workflow.get_create_opts()

            if overrides is not None:
                opts.MergeFrom(overrides)

            opts.name = name

            return self.client.PutWorkflow(
                PutWorkflowRequest(
                    opts=opts,
                ),
                metadata=get_metadata(self.token),
            )
        except grpc.RpcError as e:
            raise ValueError(f"Could not put workflow: {e}")
        
    def put_rate_limit(self, key: str, limit: int, duration: RateLimitDuration = RateLimitDuration.SECOND):
        try:
            self.client.PutRateLimit(
                PutRateLimitRequest(
                    key=key,
                    limit=limit,
                    duration=duration,
                ),
                metadata=get_metadata(self.token),
            )
        except grpc.RpcError as e:
            raise ValueError(f"Could not put rate limit: {e}")

    def schedule_workflow(self, name: str, schedules: List[Union[datetime, timestamp_pb2.Timestamp]], input={}, options: TriggerWorkflowParentOptions = None):
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
                **(options or {})
            ), metadata=get_metadata(self.token))

        except grpc.RpcError as e:
            raise ValueError(f"gRPC error: {e}")

    def run_workflow(self, workflow_name: str, input: any, options: TriggerWorkflowParentOptions = None):
        try:
            payload_data = json.dumps(input)

            resp: TriggerWorkflowResponse = self.client.TriggerWorkflow(TriggerWorkflowRequest(
                name=workflow_name,
                input=payload_data,
                **(options or {})
            ), metadata=get_metadata(self.token))

            return resp.workflow_run_id
        except grpc.RpcError as e:
            raise ValueError(f"gRPC error: {e}")
        except json.JSONDecodeError as e:
            raise ValueError(f"Error encoding payload: {e}")
