import asyncio
import json
from collections.abc import Generator
from datetime import datetime
from enum import Enum
from typing import TypeVar, cast

import grpc
from google.protobuf import timestamp_pb2
from pydantic import BaseModel, ConfigDict, Field, field_validator

from hatchet_sdk.clients.listeners.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.listeners.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts import workflows_pb2 as v0_workflow_protos
from hatchet_sdk.contracts.v1 import workflows_pb2 as workflow_protos
from hatchet_sdk.contracts.v1.workflows_pb2_grpc import AdminServiceStub
from hatchet_sdk.contracts.workflows_pb2_grpc import WorkflowServiceStub
from hatchet_sdk.exceptions import DedupeViolationError
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.rate_limit import RateLimitDuration
from hatchet_sdk.runnables.contextvars import (
    ctx_action_key,
    ctx_additional_metadata,
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


class RunStatus(str, Enum):
    QUEUED = "QUEUED"
    RUNNING = "RUNNING"
    COMPLETED = "COMPLETED"
    CANCELLED = "CANCELLED"
    FAILED = "FAILED"

    @staticmethod
    def from_proto(proto_status: workflow_protos.RunStatus) -> "RunStatus":
        if proto_status == workflow_protos.RunStatus.COMPLETED:
            return RunStatus.COMPLETED
        if proto_status == workflow_protos.RunStatus.CANCELLED:
            return RunStatus.CANCELLED
        if proto_status == workflow_protos.RunStatus.FAILED:
            return RunStatus.FAILED
        if proto_status == workflow_protos.RunStatus.RUNNING:
            return RunStatus.RUNNING
        if proto_status == workflow_protos.RunStatus.QUEUED:
            return RunStatus.QUEUED
        raise ValueError(f"Unknown proto status: {proto_status}")

    @staticmethod
    def from_v1_task_status(
        v1_task_status: V1TaskStatus,
    ) -> "RunStatus":
        if v1_task_status == V1TaskStatus.COMPLETED:
            return RunStatus.COMPLETED
        if v1_task_status == V1TaskStatus.CANCELLED:
            return RunStatus.CANCELLED
        if v1_task_status == V1TaskStatus.FAILED:
            return RunStatus.FAILED
        if v1_task_status == V1TaskStatus.RUNNING:
            return RunStatus.RUNNING
        if v1_task_status == V1TaskStatus.QUEUED:
            return RunStatus.QUEUED

        raise ValueError(f"Unknown V1TaskStatus: {v1_task_status}")

    def to_v1_task_status(self) -> V1TaskStatus:
        if self == RunStatus.COMPLETED:
            return V1TaskStatus.COMPLETED
        if self == RunStatus.CANCELLED:
            return V1TaskStatus.CANCELLED
        if self == RunStatus.FAILED:
            return V1TaskStatus.FAILED
        if self == RunStatus.RUNNING:
            return V1TaskStatus.RUNNING
        if self == RunStatus.QUEUED:
            return V1TaskStatus.QUEUED

        raise ValueError(f"Unknown RunStatus: {self}")


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


class TaskRunDetail(BaseModel):
    external_id: str
    readable_id: str
    output: JSONSerializableMapping | None = None
    error: str | None = None
    status: V1TaskStatus


class WorkflowRunDetail(BaseModel):
    external_id: str
    status: RunStatus
    input: JSONSerializableMapping | None = None
    additional_metadata: JSONSerializableMapping | None = None
    task_runs: dict[str, TaskRunDetail]
    done: bool = False


class AdminClient:
    def __init__(
        self,
        config: ClientConfig,
        workflow_run_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
    ):
        self.config = config
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

    async def aio_put_workflow(
        self,
        workflow: workflow_protos.CreateWorkflowVersionRequest,
    ) -> workflow_protos.CreateWorkflowVersionResponse:
        return await asyncio.to_thread(self.put_workflow, workflow)

    async def aio_put_rate_limit(
        self,
        key: str,
        limit: int,
        duration: RateLimitDuration = RateLimitDuration.SECOND,
    ) -> None:
        return await asyncio.to_thread(self.put_rate_limit, key, limit, duration)

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

    def put_workflow(
        self,
        workflow: workflow_protos.CreateWorkflowVersionRequest,
    ) -> workflow_protos.CreateWorkflowVersionResponse:
        if self.client is None:
            conn = new_conn(self.config, False)
            self.client = AdminServiceStub(conn)

        put_workflow = tenacity_retry(self.client.PutWorkflow, self.config.tenacity)
        return cast(
            workflow_protos.CreateWorkflowVersionResponse,
            put_workflow(
                workflow,
                metadata=get_metadata(self.token),
            ),
        )

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
        put_rate_limit = tenacity_retry(client.PutRateLimit, self.config.tenacity)

        put_rate_limit(
            v0_workflow_protos.PutRateLimitRequest(
                key=key,
                limit=limit,
                duration=duration_proto,  # type: ignore[arg-type]
            ),
            metadata=get_metadata(self.token),
        )

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
            schedule_workflow = tenacity_retry(
                client.ScheduleWorkflow, self.config.tenacity
            )

            return cast(
                v0_workflow_protos.WorkflowVersion,
                schedule_workflow(
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
        additional_metadata = ctx_additional_metadata.get() or {}
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
            additional_metadata={**additional_metadata, **options.additional_metadata},
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
    def run_workflow(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        request = self._create_workflow_run_request(workflow_name, input, options)
        client = self._get_or_create_v0_client()
        trigger_workflow = tenacity_retry(client.TriggerWorkflow, self.config.tenacity)

        try:
            resp = cast(
                v0_workflow_protos.TriggerWorkflowResponse,
                trigger_workflow(
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
            admin_client=self,
        )

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    async def aio_run_workflow(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        client = self._get_or_create_v0_client()
        trigger_workflow = tenacity_retry(client.TriggerWorkflow, self.config.tenacity)
        async with spawn_index_lock:
            request = self._create_workflow_run_request(workflow_name, input, options)

        try:
            resp = cast(
                v0_workflow_protos.TriggerWorkflowResponse,
                await asyncio.to_thread(
                    trigger_workflow,
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
            admin_client=self,
        )

    def chunk(self, xs: list[T], n: int) -> Generator[list[T], None, None]:
        for i in range(0, len(xs), n):
            yield xs[i : i + n]

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
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
        bulk_trigger_workflow = tenacity_retry(
            client.BulkTriggerWorkflow, self.config.tenacity
        )

        refs: list[WorkflowRunRef] = []

        for chunk in self.chunk(bulk_workflows, MAX_BULK_WORKFLOW_RUN_BATCH_SIZE):
            bulk_request = v0_workflow_protos.BulkTriggerWorkflowRequest(
                workflows=chunk
            )

            resp = cast(
                v0_workflow_protos.BulkTriggerWorkflowResponse,
                bulk_trigger_workflow(
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
                        admin_client=self,
                    )
                    for workflow_run_id in resp.workflow_run_ids
                ]
            )

        return refs

    async def aio_run_workflows(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[WorkflowRunRef]:
        client = self._get_or_create_v0_client()
        chunks = self.chunk(workflows, MAX_BULK_WORKFLOW_RUN_BATCH_SIZE)
        refs: list[WorkflowRunRef] = []
        bulk_trigger_workflow = tenacity_retry(
            client.BulkTriggerWorkflow, self.config.tenacity
        )

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
                    bulk_trigger_workflow,
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
                        admin_client=self,
                    )
                    for workflow_run_id in resp.workflow_run_ids
                ]
            )

        return refs

    def get_workflow_run(self, workflow_run_id: str) -> WorkflowRunRef:
        return WorkflowRunRef(
            admin_client=self,
            workflow_run_id=workflow_run_id,
            workflow_run_event_listener=self.workflow_run_event_listener,
            workflow_run_listener=self.workflow_run_listener,
        )

    def get_details(self, external_id: str) -> WorkflowRunDetail:
        if self.client is None:
            conn = new_conn(self.config, False)
            self.client = AdminServiceStub(conn)

        get_run_payloads = tenacity_retry(
            self.client.GetRunDetails, self.config.tenacity
        )

        response = cast(
            workflow_protos.GetRunDetailsResponse,
            get_run_payloads(
                workflow_protos.GetRunDetailsRequest(external_id=external_id),
                metadata=get_metadata(self.token),
            ),
        )

        return WorkflowRunDetail(
            external_id=external_id,
            input=(
                json.loads(response.input.decode("utf-8")) if response.input else None
            ),
            additional_metadata=(
                json.loads(response.additional_metadata.decode("utf-8"))
                if response.additional_metadata
                else None
            ),
            status=RunStatus.from_proto(response.status),
            task_runs={
                readable_id: TaskRunDetail(
                    readable_id=readable_id,
                    external_id=details.external_id,
                    output=(
                        json.loads(details.output.decode("utf-8"))
                        if details.output
                        else None
                    ),
                    error=details.error if details.error else None,
                    status=RunStatus.from_proto(details.status).to_v1_task_status(),
                )
                for readable_id, details in response.task_runs.items()
            },
            done=response.done,
        )
