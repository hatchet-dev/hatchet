import asyncio
import json
from datetime import datetime
from typing import Any, Union, cast

import grpc
from google.protobuf import timestamp_pb2
from pydantic import BaseModel, Field

from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.workflows_pb2 import (
    BulkTriggerWorkflowRequest,
    BulkTriggerWorkflowResponse,
    CreateWorkflowVersionOpts,
    PutRateLimitRequest,
    PutWorkflowRequest,
    RateLimitDuration,
    ScheduleWorkflowRequest,
    TriggerWorkflowRequest,
    TriggerWorkflowResponse,
    WorkflowVersion,
)
from hatchet_sdk.contracts.workflows_pb2_grpc import WorkflowServiceStub
from hatchet_sdk.loader import ClientConfig
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.utils.types import JSONSerializableDict
from hatchet_sdk.workflow_run import WorkflowRunRef


class ScheduleTriggerWorkflowOptions(BaseModel):
    parent_id: str | None = None
    parent_step_run_id: str | None = None
    child_index: int | None = None
    child_key: str | None = None
    namespace: str | None = None


class ChildTriggerWorkflowOptions(BaseModel):
    additional_metadata: JSONSerializableDict = Field(default_factory=dict)
    sticky: bool = False


class ChildWorkflowRunDict(BaseModel):
    workflow_name: str
    input: JSONSerializableDict
    options: ChildTriggerWorkflowOptions
    key: str | None = None


class TriggerWorkflowOptions(ScheduleTriggerWorkflowOptions):
    additional_metadata: JSONSerializableDict = Field(default_factory=dict)
    desired_worker_id: str | None = None
    namespace: str | None = None


class WorkflowRunDict(BaseModel):
    workflow_name: str
    input: JSONSerializableDict
    options: TriggerWorkflowOptions


class DedupeViolationErr(Exception):
    """Raised by the Hatchet library to indicate that a workflow has already been run with this deduplication value."""

    pass


class AdminClient:
    def __init__(self, config: ClientConfig):
        conn = new_conn(config, False)
        self.config = config
        self.client = WorkflowServiceStub(conn)  # type: ignore[no-untyped-call]
        self.token = config.token
        self.listener_client = RunEventListenerClient(config=config)
        self.namespace = config.namespace

        self.pooled_workflow_listener: PooledWorkflowRunListener | None = None

    def _prepare_workflow_request(
        self, workflow_name: str, input: dict[str, Any], options: TriggerWorkflowOptions
    ) -> TriggerWorkflowRequest:
        try:
            payload_data = json.dumps(input)
            _options = options.model_dump()

            _options.pop("namespace")

            try:
                _options = {
                    **_options,
                    "additional_metadata": json.dumps(
                        options.additional_metadata
                    ).encode("utf-8"),
                }
            except json.JSONDecodeError as e:
                raise ValueError(f"Error encoding payload: {e}")

            return TriggerWorkflowRequest(
                name=workflow_name, input=payload_data, **_options
            )
        except json.JSONDecodeError as e:
            raise ValueError(f"Error encoding payload: {e}")

    def _prepare_put_workflow_request(
        self,
        name: str,
        workflow: CreateWorkflowVersionOpts,
        overrides: CreateWorkflowVersionOpts | None = None,
    ) -> PutWorkflowRequest:
        if overrides is not None:
            workflow.MergeFrom(overrides)

        workflow.name = name

        return PutWorkflowRequest(
            opts=workflow,
        )

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
        input: JSONSerializableDict = {},
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> ScheduleWorkflowRequest:
        return ScheduleWorkflowRequest(
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
        input: JSONSerializableDict,
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
        workflows: list[WorkflowRunDict],
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
        workflow: CreateWorkflowVersionOpts,
        overrides: CreateWorkflowVersionOpts | None = None,
    ) -> WorkflowVersion:
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
        input: JSONSerializableDict = {},
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> WorkflowVersion:
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
        workflow: CreateWorkflowVersionOpts,
        overrides: CreateWorkflowVersionOpts | None = None,
    ) -> WorkflowVersion:
        opts = self._prepare_put_workflow_request(name, workflow, overrides)

        resp: WorkflowVersion = self.client.PutWorkflow(
            opts,
            metadata=get_metadata(self.token),
        )

        return resp

    @tenacity_retry
    def put_rate_limit(
        self,
        key: str,
        limit: int,
        duration: Union[RateLimitDuration, str] = RateLimitDuration.SECOND,
    ) -> None:
        self.client.PutRateLimit(
            PutRateLimitRequest(
                key=key,
                limit=limit,
                duration=duration,
            ),
            metadata=get_metadata(self.token),
        )

    @tenacity_retry
    def schedule_workflow(
        self,
        name: str,
        schedules: list[Union[datetime, timestamp_pb2.Timestamp]],
        input: JSONSerializableDict = {},
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> WorkflowVersion:
        try:
            namespace = options.namespace or self.namespace

            if namespace != "" and not name.startswith(self.namespace):
                name = f"{namespace}{name}"

            request = self._prepare_schedule_workflow_request(
                name, schedules, input, options
            )

            return cast(
                WorkflowVersion,
                self.client.ScheduleWorkflow(
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
        input: JSONSerializableDict,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        try:
            if not self.pooled_workflow_listener:
                self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

            namespace = options.namespace or self.namespace

            if namespace != "" and not workflow_name.startswith(self.namespace):
                workflow_name = f"{namespace}{workflow_name}"

            request = self._prepare_workflow_request(workflow_name, input, options)

            resp = cast(
                TriggerWorkflowResponse,
                self.client.TriggerWorkflow(
                    request,
                    metadata=get_metadata(self.token),
                ),
            )

            return WorkflowRunRef(
                workflow_run_id=resp.workflow_run_id,
                workflow_listener=self.pooled_workflow_listener,
                workflow_run_event_listener=self.listener_client,
            )
        except (grpc.RpcError, grpc.aio.AioRpcError) as e:
            if e.code() == grpc.StatusCode.ALREADY_EXISTS:
                raise DedupeViolationErr(e.details())

            raise e

    def _prepare_workflow_run_request(
        self, workflow: WorkflowRunDict, options: TriggerWorkflowOptions
    ) -> TriggerWorkflowRequest:
        workflow_name = workflow.workflow_name
        input_data = workflow.input
        options = workflow.options

        namespace = options.namespace or self.namespace

        if namespace != "" and not workflow_name.startswith(self.namespace):
            workflow_name = f"{namespace}{workflow_name}"

        return self._prepare_workflow_request(workflow_name, input_data, options)

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    def run_workflows(
        self,
        workflows: list[WorkflowRunDict],
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> list[WorkflowRunRef]:
        if not self.pooled_workflow_listener:
            self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

        bulk_request = BulkTriggerWorkflowRequest(
            workflows=[
                self._prepare_workflow_run_request(workflow, options)
                for workflow in workflows
            ]
        )

        resp: BulkTriggerWorkflowResponse = self.client.BulkTriggerWorkflow(
            bulk_request,
            metadata=get_metadata(self.token),
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
