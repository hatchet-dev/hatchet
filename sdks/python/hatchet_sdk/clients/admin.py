import asyncio
import json
from datetime import datetime
from typing import Union, cast

import grpc
from google.protobuf import timestamp_pb2
from pydantic import BaseModel, ConfigDict, Field, field_validator

from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts import workflows_pb2 as v0_workflow_protos
from hatchet_sdk.contracts.v1 import workflows_pb2 as workflow_protos
from hatchet_sdk.contracts.v1.workflows_pb2_grpc import AdminServiceStub
from hatchet_sdk.contracts.workflows_pb2_grpc import WorkflowServiceStub
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.rate_limit import RateLimitDuration
from hatchet_sdk.utils.proto_enums import convert_python_enum_to_proto, maybe_int_to_str
from hatchet_sdk.utils.typing import JSONSerializableMapping
from hatchet_sdk.worker.action_listener_process import (
    ctx_spawn_index,
    ctx_step_run_id,
    ctx_worker_id,
    ctx_workflow_run_id,
)
from hatchet_sdk.workflow_run import WorkflowRunRef


class ScheduleTriggerWorkflowOptions(BaseModel):
    parent_id: str | None = None
    parent_step_run_id: str | None = None
    child_index: int | None = None
    child_key: str | None = None
    namespace: str | None = None


class TriggerWorkflowOptions(ScheduleTriggerWorkflowOptions):
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)
    desired_worker_id: str | None = None
    namespace: str | None = None
    sticky: bool = False
    key: str | None = None


class WorkflowRunTriggerConfig(BaseModel):
    workflow_name: str
    input: JSONSerializableMapping
    options: TriggerWorkflowOptions
    key: str | None = None


class DedupeViolationErr(Exception):
    """Raised by the Hatchet library to indicate that a workflow has already been run with this deduplication value."""

    pass


class AdminClient:
    def __init__(self, config: ClientConfig):
        conn = new_conn(config, False)
        self.config = config
        self.client = AdminServiceStub(conn)  # type: ignore[no-untyped-call]
        self.v0_client = WorkflowServiceStub(conn)  # type: ignore[no-untyped-call]
        self.token = config.token
        self.listener_client = RunEventListenerClient(config=config)
        self.namespace = config.namespace

        self.pooled_workflow_listener: PooledWorkflowRunListener | None = None

    class TriggerWorkflowRequest(BaseModel):
        model_config = ConfigDict(extra="ignore")

        parent_id: str | None = None
        parent_step_run_id: str | None = None
        child_index: int | None = None
        child_key: str | None = None
        additional_metadata: str | None = None
        desired_worker_id: str | None = None
        priority: int | None = None

        @field_validator("additional_metadata", mode="before")
        @classmethod
        def validate_additional_metadata(
            cls, v: JSONSerializableMapping | None
        ) -> bytes | None:
            if not v:
                return None

            try:
                return json.dumps(v).encode("utf-8")
            except json.JSONDecodeError as e:
                raise ValueError(f"Error encoding payload: {e}")

    def _prepare_workflow_request(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        options: TriggerWorkflowOptions,
    ) -> workflow_protos.TriggerWorkflowRunRequest:
        try:
            payload_data = json.dumps(input).encode("utf-8")
        except json.JSONDecodeError as e:
            raise ValueError(f"Error encoding payload: {e}")

        _options = self.TriggerWorkflowRequest.model_validate(
            options.model_dump()
        ).model_dump()

        return workflow_protos.TriggerWorkflowRunRequest(
            workflow_name=workflow_name, input=payload_data, **_options
        )

    def _prepare_put_workflow_request(
        self,
        name: str,
        workflow: workflow_protos.CreateWorkflowVersionRequest,
        overrides: workflow_protos.CreateWorkflowVersionRequest | None = None,
    ) -> workflow_protos.CreateWorkflowVersionRequest:
        if overrides is not None:
            workflow.MergeFrom(overrides)

        workflow.name = name

        return workflow

    def _parse_schedule(
        self, schedule: datetime | timestamp_pb2.Timestamp
    ) -> timestamp_pb2.Timestamp:
        if isinstance(schedule, datetime):
            t = schedule.timestamp()
            seconds = int(t)
            nanos = int(t % 1 * 1e9)
            return timestamp_pb2.Timestamp(seconds=seconds, nanos=nanos)
        elif isinstance(schedule, timestamp_pb2.Timestamp):
            return schedule
        else:
            raise ValueError(
                "Invalid schedule type. Must be datetime or timestamp_pb2.Timestamp."
            )

    def _prepare_schedule_workflow_request(
        self,
        name: str,
        schedules: list[Union[datetime, timestamp_pb2.Timestamp]],
        input: JSONSerializableMapping = {},
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> v0_workflow_protos.ScheduleWorkflowRequest:
        return v0_workflow_protos.ScheduleWorkflowRequest(
            name=name,
            schedules=[self._parse_schedule(schedule) for schedule in schedules],
            input=json.dumps(input),
            **options.model_dump(),
        )

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    async def aio_run_workflow(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        ## IMPORTANT: The `pooled_workflow_listener` must be created 1) lazily, and not at `init` time, and 2) on the
        ## main thread. If 1) is not followed, you'll get an error about something being attached to the wrong event
        ## loop. If 2) is not followed, you'll get an error about the event loop not being set up.
        if not self.pooled_workflow_listener:
            self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

        return await asyncio.to_thread(self.run_workflow, workflow_name, input, options)

    @tenacity_retry
    async def aio_run_workflows(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> list[WorkflowRunRef]:
        ## IMPORTANT: The `pooled_workflow_listener` must be created 1) lazily, and not at `init` time, and 2) on the
        ## main thread. If 1) is not followed, you'll get an error about something being attached to the wrong event
        ## loop. If 2) is not followed, you'll get an error about the event loop not being set up.
        if not self.pooled_workflow_listener:
            self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

        return await asyncio.to_thread(self.run_workflows, workflows, options)

    @tenacity_retry
    async def aio_put_workflow(
        self,
        name: str,
        workflow: workflow_protos.CreateWorkflowVersionRequest,
        overrides: workflow_protos.CreateWorkflowVersionRequest | None = None,
    ) -> workflow_protos.CreateWorkflowVersionResponse:
        ## IMPORTANT: The `pooled_workflow_listener` must be created 1) lazily, and not at `init` time, and 2) on the
        ## main thread. If 1) is not followed, you'll get an error about something being attached to the wrong event
        ## loop. If 2) is not followed, you'll get an error about the event loop not being set up.
        if not self.pooled_workflow_listener:
            self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

        return await asyncio.to_thread(self.put_workflow, name, workflow, overrides)

    @tenacity_retry
    async def aio_put_rate_limit(
        self,
        key: str,
        limit: int,
        duration: RateLimitDuration = RateLimitDuration.SECOND,
    ) -> None:
        ## IMPORTANT: The `pooled_workflow_listener` must be created 1) lazily, and not at `init` time, and 2) on the
        ## main thread. If 1) is not followed, you'll get an error about something being attached to the wrong event
        ## loop. If 2) is not followed, you'll get an error about the event loop not being set up.
        if not self.pooled_workflow_listener:
            self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

        return await asyncio.to_thread(self.put_rate_limit, key, limit, duration)

    @tenacity_retry
    async def aio_schedule_workflow(
        self,
        name: str,
        schedules: list[Union[datetime, timestamp_pb2.Timestamp]],
        input: JSONSerializableMapping = {},
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> v0_workflow_protos.WorkflowVersion:
        ## IMPORTANT: The `pooled_workflow_listener` must be created 1) lazily, and not at `init` time, and 2) on the
        ## main thread. If 1) is not followed, you'll get an error about something being attached to the wrong event
        ## loop. If 2) is not followed, you'll get an error about the event loop not being set up.
        if not self.pooled_workflow_listener:
            self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

        return await asyncio.to_thread(
            self.schedule_workflow, name, schedules, input, options
        )

    @tenacity_retry
    def put_workflow(
        self,
        name: str,
        workflow: workflow_protos.CreateWorkflowVersionRequest,
        overrides: workflow_protos.CreateWorkflowVersionRequest | None = None,
    ) -> workflow_protos.CreateWorkflowVersionResponse:
        opts = self._prepare_put_workflow_request(name, workflow, overrides)

        print("\n\nPutting workflow", opts, get_metadata(self.token))

        return cast(
            workflow_protos.CreateWorkflowVersionResponse,
            self.client.PutWorkflow(
                opts,
                metadata=get_metadata(self.token),
            ),
        )

    @tenacity_retry
    def put_rate_limit(
        self,
        key: str,
        limit: int,
        duration: RateLimitDuration = RateLimitDuration.SECOND,
    ) -> None:
        duration_proto = convert_python_enum_to_proto(
            duration, workflow_protos.RateLimitDuration
        )
        self.v0_client.PutRateLimit(
            v0_workflow_protos.PutRateLimitRequest(
                key=key,
                limit=limit,
                duration=maybe_int_to_str(duration_proto),
            ),
            metadata=get_metadata(self.token),
        )

    @tenacity_retry
    def schedule_workflow(
        self,
        name: str,
        schedules: list[Union[datetime, timestamp_pb2.Timestamp]],
        input: JSONSerializableMapping = {},
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> v0_workflow_protos.WorkflowVersion:
        try:
            namespace = options.namespace or self.namespace

            if namespace != "" and not name.startswith(self.namespace):
                name = f"{namespace}{name}"

            request = self._prepare_schedule_workflow_request(
                name, schedules, input, options
            )

            return cast(
                v0_workflow_protos.WorkflowVersion,
                self.v0_client.ScheduleWorkflow(
                    request,
                    metadata=get_metadata(self.token),
                ),
            )
        except (grpc.RpcError, grpc.aio.AioRpcError) as e:
            if e.code() == grpc.StatusCode.ALREADY_EXISTS:
                raise DedupeViolationErr(e.details())

            raise e

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    def run_workflow(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        workflow_run_id = ctx_workflow_run_id.get()
        step_run_id = ctx_step_run_id.get()
        worker_id = ctx_worker_id.get()
        spawn_index = ctx_spawn_index.get()

        desired_worker_id = (
            (options.desired_worker_id or worker_id) if options.sticky else None
        )

        trigger_options = TriggerWorkflowOptions(
            parent_id=options.parent_id or workflow_run_id,
            parent_step_run_id=options.parent_step_run_id or step_run_id,
            child_key=options.child_key,
            child_index=spawn_index,
            additional_metadata=options.additional_metadata,
            desired_worker_id=desired_worker_id,
        )

        ## TODO: I think this isn't safe to do b/c of state update races causing these
        ## updates to not be atomic
        ctx_spawn_index.set(spawn_index + 1)

        try:
            if not self.pooled_workflow_listener:
                self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

            namespace = options.namespace or self.namespace

            if namespace != "" and not workflow_name.startswith(self.namespace):
                workflow_name = f"{namespace}{workflow_name}"

            request = self._prepare_workflow_request(
                workflow_name, input, trigger_options
            )

            resp = cast(
                workflow_protos.TriggerWorkflowRunResponse,
                self.client.TriggerWorkflowRun(
                    request,
                    metadata=get_metadata(self.token),
                ),
            )

            return WorkflowRunRef(
                workflow_run_id=resp.external_id,
                workflow_listener=self.pooled_workflow_listener,
                workflow_run_event_listener=self.listener_client,
            )
        except (grpc.RpcError, grpc.aio.AioRpcError) as e:
            if e.code() == grpc.StatusCode.ALREADY_EXISTS:
                raise DedupeViolationErr(e.details())

            raise e

    def _prepare_workflow_run_request(
        self, workflow: WorkflowRunTriggerConfig, options: TriggerWorkflowOptions
    ) -> workflow_protos.TriggerWorkflowRunRequest:
        workflow_name = workflow.workflow_name
        input_data = workflow.input
        options = workflow.options

        spawn_index = ctx_spawn_index.get()

        desired_worker_id = (
            (options.desired_worker_id or ctx_worker_id.get())
            if options.sticky
            else None
        )

        options.parent_id = options.parent_id or ctx_workflow_run_id.get()
        options.parent_step_run_id = options.parent_step_run_id or ctx_step_run_id.get()
        options.child_index = spawn_index
        options.desired_worker_id = desired_worker_id

        ctx_spawn_index.set(spawn_index + 1)

        namespace = options.namespace or self.namespace

        if namespace != "" and not workflow_name.startswith(self.namespace):
            workflow_name = f"{namespace}{workflow_name}"

        return self._prepare_workflow_request(workflow_name, input_data, options)

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    def run_workflows(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> list[WorkflowRunRef]:
        if not self.pooled_workflow_listener:
            self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

        bulk_request = v0_workflow_protos.BulkTriggerWorkflowRequest(
            workflows=[
                self._prepare_workflow_run_request(workflow, options)
                for workflow in workflows
            ]
        )

        resp = cast(
            v0_workflow_protos.BulkTriggerWorkflowResponse,
            self.v0_client.BulkTriggerWorkflow(
                bulk_request,
                metadata=get_metadata(self.token),
            ),
        )

        return [
            WorkflowRunRef(
                workflow_run_id=workflow_run_id,
                workflow_listener=self.pooled_workflow_listener,
                workflow_run_event_listener=self.listener_client,
            )
            for workflow_run_id in resp.workflow_run_ids
        ]

    def get_workflow_run(self, workflow_run_id: str) -> WorkflowRunRef:
        if not self.pooled_workflow_listener:
            self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

        return WorkflowRunRef(
            workflow_run_id=workflow_run_id,
            workflow_listener=self.pooled_workflow_listener,
            workflow_run_event_listener=self.listener_client,
        )
