import asyncio
import json
from collections.abc import Generator
from datetime import datetime
from typing import TypeVar, cast

import grpc
from google.protobuf import timestamp_pb2
from pydantic import BaseModel, ConfigDict, Field, field_validator

from hatchet_sdk.clients.listeners.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.listeners.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts import workflows_pb2 as v0_workflow_protos
from hatchet_sdk.contracts.v1 import workflows_pb2 as workflow_protos
from hatchet_sdk.contracts.v1.workflows_pb2_grpc import AdminServiceStub
from hatchet_sdk.contracts.workflows_pb2_grpc import WorkflowServiceStub
from hatchet_sdk.exceptions import DedupeViolationError
from hatchet_sdk.features.runs import RunsClient
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.rate_limit import RateLimitDuration
from hatchet_sdk.runnables.contextvars import (
    ctx_action_key,
    ctx_step_run_id,
    ctx_worker_id,
    ctx_workflow_run_id,
    spawn_index_lock,
    workflow_spawn_indices,
)
from hatchet_sdk.utils.proto_enums import convert_python_enum_to_proto
from hatchet_sdk.utils.typing import JSONSerializableMapping
from hatchet_sdk.workflow_run import WorkflowRunRef

T = TypeVar("T")

MAX_BULK_WORKFLOW_RUN_BATCH_SIZE = 1000


class ScheduleTriggerWorkflowOptions(BaseModel):
    parent_id: str | None = None
    parent_step_run_id: str | None = None
    child_index: int | None = None
    child_key: str | None = None
    namespace: str | None = None
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)
    priority: int | None = None


class TriggerWorkflowOptions(ScheduleTriggerWorkflowOptions):
    desired_worker_id: str | None = None
    sticky: bool = False
    key: str | None = None


class WorkflowRunTriggerConfig(BaseModel):
    workflow_name: str
    input: JSONSerializableMapping
    options: TriggerWorkflowOptions
    key: str | None = None


class AdminClient:
    def __init__(
        self,
        config: ClientConfig,
        workflow_run_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
        runs_client: RunsClient,
    ):
        self.config = config
        self.runs_client = runs_client
        self.token = config.token
        self.namespace = config.namespace

        self.workflow_run_listener = workflow_run_listener
        self.workflow_run_event_listener = workflow_run_event_listener

        self.client: AdminServiceStub | None = None
        self.v0_client: WorkflowServiceStub | None = None

    def _get_or_create_v0_client(self) -> WorkflowServiceStub:
        if self.v0_client is None:
            conn = new_conn(self.config, False)
            self.v0_client = WorkflowServiceStub(conn)

        return self.v0_client

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
                raise ValueError("Error encoding payload") from e

    def _prepare_workflow_request(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        options: TriggerWorkflowOptions,
    ) -> v0_workflow_protos.TriggerWorkflowRequest:
        try:
            payload_data = json.dumps(input)
        except json.JSONDecodeError as e:
            raise ValueError("Error encoding payload") from e

        _options = self.TriggerWorkflowRequest.model_validate(options.model_dump())

        return v0_workflow_protos.TriggerWorkflowRequest(
            name=workflow_name,
            input=payload_data,
            parent_id=_options.parent_id,
            parent_step_run_id=_options.parent_step_run_id,
            child_index=_options.child_index,
            child_key=_options.child_key,
            additional_metadata=_options.additional_metadata,
            desired_worker_id=_options.desired_worker_id,
            priority=_options.priority,
        )

    def _parse_schedule(
        self, schedule: datetime | timestamp_pb2.Timestamp
    ) -> timestamp_pb2.Timestamp:
        if isinstance(schedule, datetime):
            t = schedule.timestamp()
            seconds = int(t)
            nanos = int(t % 1 * 1e9)
            return timestamp_pb2.Timestamp(seconds=seconds, nanos=nanos)
        if isinstance(schedule, timestamp_pb2.Timestamp):
            return schedule
        raise ValueError(
            "Invalid schedule type. Must be datetime or timestamp_pb2.Timestamp."
        )

    def _prepare_schedule_workflow_request(
        self,
        name: str,
        schedules: list[datetime | timestamp_pb2.Timestamp],
        input: JSONSerializableMapping | None = None,
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> v0_workflow_protos.ScheduleWorkflowRequest:
        return v0_workflow_protos.ScheduleWorkflowRequest(
            name=name,
            schedules=[self._parse_schedule(schedule) for schedule in schedules],
            input=json.dumps(input),
            parent_id=options.parent_id,
            parent_step_run_id=options.parent_step_run_id,
            child_index=options.child_index,
            child_key=options.child_key,
            additional_metadata=json.dumps(options.additional_metadata),
            priority=options.priority,
        )

    @tenacity_retry
    async def aio_put_workflow(
        self,
        workflow: workflow_protos.CreateWorkflowVersionRequest,
    ) -> workflow_protos.CreateWorkflowVersionResponse:
        return await asyncio.to_thread(self.put_workflow, workflow)

    @tenacity_retry
    async def aio_put_rate_limit(
        self,
        key: str,
        limit: int,
        duration: RateLimitDuration = RateLimitDuration.SECOND,
    ) -> None:
        return await asyncio.to_thread(self.put_rate_limit, key, limit, duration)

    @tenacity_retry
    async def aio_schedule_workflow(
        self,
        name: str,
        schedules: list[datetime | timestamp_pb2.Timestamp],
        input: JSONSerializableMapping | None = None,
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> v0_workflow_protos.WorkflowVersion:
        return await asyncio.to_thread(
            self.schedule_workflow, name, schedules, input, options
        )

    @tenacity_retry
    def put_workflow(
        self,
        workflow: workflow_protos.CreateWorkflowVersionRequest,
    ) -> workflow_protos.CreateWorkflowVersionResponse:
        if self.client is None:
            conn = new_conn(self.config, False)
            self.client = AdminServiceStub(conn)

        return cast(
            workflow_protos.CreateWorkflowVersionResponse,
            self.client.PutWorkflow(
                workflow,
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

        client = self._get_or_create_v0_client()

        client.PutRateLimit(
            v0_workflow_protos.PutRateLimitRequest(
                key=key,
                limit=limit,
                duration=duration_proto,  # type: ignore[arg-type]
            ),
            metadata=get_metadata(self.token),
        )

    @tenacity_retry
    def schedule_workflow(
        self,
        name: str,
        schedules: list[datetime | timestamp_pb2.Timestamp],
        input: JSONSerializableMapping | None = None,
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> v0_workflow_protos.WorkflowVersion:
        try:
            namespace = options.namespace or self.namespace

            name = self.config.apply_namespace(name, namespace)

            request = self._prepare_schedule_workflow_request(
                name, schedules, input, options
            )

            client = self._get_or_create_v0_client()

            return cast(
                v0_workflow_protos.WorkflowVersion,
                client.ScheduleWorkflow(
                    request,
                    metadata=get_metadata(self.token),
                ),
            )
        except (grpc.RpcError, grpc.aio.AioRpcError) as e:
            if e.code() == grpc.StatusCode.ALREADY_EXISTS:
                raise DedupeViolationError(e.details()) from e

            raise e

    def _create_workflow_run_request(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        options: TriggerWorkflowOptions,
    ) -> v0_workflow_protos.TriggerWorkflowRequest:
        workflow_run_id = ctx_workflow_run_id.get()
        step_run_id = ctx_step_run_id.get()
        worker_id = ctx_worker_id.get()
        action_key = ctx_action_key.get()
        spawn_index = workflow_spawn_indices[action_key] if action_key else 0

        ## Increment the spawn_index for the parent workflow
        if action_key:
            workflow_spawn_indices[action_key] += 1

        desired_worker_id = (
            (options.desired_worker_id or worker_id) if options.sticky else None
        )
        child_index = (
            options.child_index if options.child_index is not None else spawn_index
        )

        trigger_options = TriggerWorkflowOptions(
            parent_id=options.parent_id or workflow_run_id,
            parent_step_run_id=options.parent_step_run_id or step_run_id,
            child_key=options.child_key,
            child_index=child_index,
            additional_metadata=options.additional_metadata,
            desired_worker_id=desired_worker_id,
            priority=options.priority,
            namespace=options.namespace,
            sticky=options.sticky,
            key=options.key,
        )

        namespace = options.namespace or self.namespace

        workflow_name = self.config.apply_namespace(workflow_name, namespace)

        return self._prepare_workflow_request(workflow_name, input, trigger_options)

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    def run_workflow(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        request = self._create_workflow_run_request(workflow_name, input, options)
        client = self._get_or_create_v0_client()

        try:
            resp = cast(
                v0_workflow_protos.TriggerWorkflowResponse,
                client.TriggerWorkflow(
                    request,
                    metadata=get_metadata(self.token),
                ),
            )
        except (grpc.RpcError, grpc.aio.AioRpcError) as e:
            if e.code() == grpc.StatusCode.ALREADY_EXISTS:
                raise DedupeViolationError(e.details()) from e
            raise e

        return WorkflowRunRef(
            workflow_run_id=resp.workflow_run_id,
            workflow_run_event_listener=self.workflow_run_event_listener,
            workflow_run_listener=self.workflow_run_listener,
            runs_client=self.runs_client,
        )

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    async def aio_run_workflow(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        client = self._get_or_create_v0_client()
        async with spawn_index_lock:
            request = self._create_workflow_run_request(workflow_name, input, options)

        try:
            resp = cast(
                v0_workflow_protos.TriggerWorkflowResponse,
                await asyncio.to_thread(
                    client.TriggerWorkflow,
                    request,
                    metadata=get_metadata(self.token),
                ),
            )
        except (grpc.RpcError, grpc.aio.AioRpcError) as e:
            if e.code() == grpc.StatusCode.ALREADY_EXISTS:
                raise DedupeViolationError(e.details()) from e

            raise e

        return WorkflowRunRef(
            runs_client=self.runs_client,
            workflow_run_id=resp.workflow_run_id,
            workflow_run_event_listener=self.workflow_run_event_listener,
            workflow_run_listener=self.workflow_run_listener,
        )

    def chunk(self, xs: list[T], n: int) -> Generator[list[T], None, None]:
        for i in range(0, len(xs), n):
            yield xs[i : i + n]

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    def run_workflows(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[WorkflowRunRef]:
        client = self._get_or_create_v0_client()
        bulk_workflows = [
            self._create_workflow_run_request(
                workflow.workflow_name, workflow.input, workflow.options
            )
            for workflow in workflows
        ]

        refs: list[WorkflowRunRef] = []

        for chunk in self.chunk(bulk_workflows, MAX_BULK_WORKFLOW_RUN_BATCH_SIZE):
            bulk_request = v0_workflow_protos.BulkTriggerWorkflowRequest(
                workflows=chunk
            )

            resp = cast(
                v0_workflow_protos.BulkTriggerWorkflowResponse,
                client.BulkTriggerWorkflow(
                    bulk_request,
                    metadata=get_metadata(self.token),
                ),
            )

            refs.extend(
                [
                    WorkflowRunRef(
                        workflow_run_id=workflow_run_id,
                        workflow_run_event_listener=self.workflow_run_event_listener,
                        workflow_run_listener=self.workflow_run_listener,
                        runs_client=self.runs_client,
                    )
                    for workflow_run_id in resp.workflow_run_ids
                ]
            )

        return refs

    @tenacity_retry
    async def aio_run_workflows(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[WorkflowRunRef]:
        client = self._get_or_create_v0_client()
        chunks = self.chunk(workflows, MAX_BULK_WORKFLOW_RUN_BATCH_SIZE)
        refs: list[WorkflowRunRef] = []

        for chunk in chunks:
            async with spawn_index_lock:
                bulk_workflows = [
                    self._create_workflow_run_request(
                        workflow.workflow_name, workflow.input, workflow.options
                    )
                    for workflow in chunk
                ]

            bulk_request = v0_workflow_protos.BulkTriggerWorkflowRequest(
                workflows=bulk_workflows
            )

            resp = cast(
                v0_workflow_protos.BulkTriggerWorkflowResponse,
                await asyncio.to_thread(
                    client.BulkTriggerWorkflow,
                    bulk_request,
                    metadata=get_metadata(self.token),
                ),
            )

            refs.extend(
                [
                    WorkflowRunRef(
                        workflow_run_id=workflow_run_id,
                        workflow_run_event_listener=self.workflow_run_event_listener,
                        workflow_run_listener=self.workflow_run_listener,
                        runs_client=self.runs_client,
                    )
                    for workflow_run_id in resp.workflow_run_ids
                ]
            )

        return refs

    def get_workflow_run(self, workflow_run_id: str) -> WorkflowRunRef:
        return WorkflowRunRef(
            runs_client=self.runs_client,
            workflow_run_id=workflow_run_id,
            workflow_run_event_listener=self.workflow_run_event_listener,
            workflow_run_listener=self.workflow_run_listener,
        )
