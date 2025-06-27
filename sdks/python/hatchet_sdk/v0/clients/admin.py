import json
from datetime import datetime
from typing import Any, Callable, Dict, List, Optional, TypedDict, TypeVar, Union

import grpc
from google.protobuf import timestamp_pb2

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
from hatchet_sdk.v0.clients.rest.models.workflow_run import WorkflowRun
from hatchet_sdk.v0.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.v0.clients.run_event_listener import new_listener
from hatchet_sdk.v0.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.v0.connection import new_conn
from hatchet_sdk.v0.workflow_run import RunRef, WorkflowRunRef

from ..loader import ClientConfig
from ..metadata import get_metadata
from ..workflow import WorkflowMeta


def new_admin(config: ClientConfig):
    return AdminClient(config)


class ScheduleTriggerWorkflowOptions(TypedDict, total=False):
    parent_id: Optional[str]
    parent_step_run_id: Optional[str]
    child_index: Optional[int]
    child_key: Optional[str]
    namespace: Optional[str]


class ChildTriggerWorkflowOptions(TypedDict, total=False):
    additional_metadata: Dict[str, str] | None = None
    sticky: bool | None = None


class ChildWorkflowRunDict(TypedDict, total=False):
    workflow_name: str
    input: Any
    options: ChildTriggerWorkflowOptions
    key: str | None = None


class TriggerWorkflowOptions(ScheduleTriggerWorkflowOptions, total=False):
    additional_metadata: Dict[str, str] | None = None
    desired_worker_id: str | None = None
    namespace: str | None = None


class WorkflowRunDict(TypedDict, total=False):
    workflow_name: str
    input: Any
    options: TriggerWorkflowOptions | None


class DedupeViolationErr(Exception):
    """Raised by the Hatchet library to indicate that a workflow has already been run with this deduplication value."""

    pass


class AdminClientBase:
    pooled_workflow_listener: PooledWorkflowRunListener | None = None

    def _prepare_workflow_request(
        self, workflow_name: str, input: any, options: TriggerWorkflowOptions = None
    ):
        try:
            payload_data = json.dumps(input)

            try:
                meta = (
                    None
                    if options is None or "additional_metadata" not in options
                    else options["additional_metadata"]
                )
                if meta is not None:
                    options = {
                        **options,
                        "additional_metadata": json.dumps(meta).encode("utf-8"),
                    }
            except json.JSONDecodeError as e:
                raise ValueError(f"Error encoding payload: {e}")

            return TriggerWorkflowRequest(
                name=workflow_name, input=payload_data, **(options or {})
            )
        except json.JSONDecodeError as e:
            raise ValueError(f"Error encoding payload: {e}")

    def _prepare_put_workflow_request(
        self,
        name: str,
        workflow: CreateWorkflowVersionOpts | WorkflowMeta,
        overrides: CreateWorkflowVersionOpts | None = None,
    ):
        try:
            opts: CreateWorkflowVersionOpts

            if isinstance(workflow, CreateWorkflowVersionOpts):
                opts = workflow
            else:
                opts = workflow.get_create_opts(self.client.config.namespace)

            if overrides is not None:
                opts.MergeFrom(overrides)

            opts.name = name

            return PutWorkflowRequest(
                opts=opts,
            )
        except grpc.RpcError as e:
            raise ValueError(f"Could not put workflow: {e}")

    def _prepare_schedule_workflow_request(
        self,
        name: str,
        schedules: List[Union[datetime, timestamp_pb2.Timestamp]],
        input={},
        options: ScheduleTriggerWorkflowOptions = None,
    ):
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
                raise ValueError(
                    "Invalid schedule type. Must be datetime or timestamp_pb2.Timestamp."
                )

        return ScheduleWorkflowRequest(
            name=name,
            schedules=timestamp_schedules,
            input=json.dumps(input),
            **(options or {}),
        )


T = TypeVar("T")


class AdminClientAioImpl(AdminClientBase):
    def __init__(self, config: ClientConfig):
        aio_conn = new_conn(config, True)
        self.config = config
        self.aio_client = WorkflowServiceStub(aio_conn)
        self.token = config.token
        self.listener_client = new_listener(config)
        self.namespace = config.namespace

    async def run(
        self,
        function: Union[str, Callable[[Any], T]],
        input: any,
        options: TriggerWorkflowOptions = None,
    ) -> "RunRef[T]":
        workflow_name = function

        if not isinstance(function, str):
            workflow_name = function.function_name

        wrr = await self.run_workflow(workflow_name, input, options)

        return RunRef[T](
            wrr.workflow_run_id, wrr.workflow_listener, wrr.workflow_run_event_listener
        )

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    async def run_workflow(
        self, workflow_name: str, input: any, options: TriggerWorkflowOptions = None
    ) -> WorkflowRunRef:
        try:
            if not self.pooled_workflow_listener:
                self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

            namespace = self.namespace

            if (
                options is not None
                and "namespace" in options
                and options["namespace"] is not None
            ):
                namespace = options.pop("namespace")

            if namespace != "" and not workflow_name.startswith(self.namespace):
                workflow_name = f"{namespace}{workflow_name}"

            request = self._prepare_workflow_request(workflow_name, input, options)

            resp: TriggerWorkflowResponse = await self.aio_client.TriggerWorkflow(
                request,
                metadata=get_metadata(self.token),
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

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    async def run_workflows(
        self,
        workflows: list[WorkflowRunDict],
        options: TriggerWorkflowOptions | None = None,
    ) -> List[WorkflowRunRef]:
        if len(workflows) == 0:
            raise ValueError("No workflows to run")

        if not self.pooled_workflow_listener:
            self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

        namespace = self.namespace

        if (
            options is not None
            and "namespace" in options
            and options["namespace"] is not None
        ):
            namespace = options["namespace"]
            del options["namespace"]

        workflow_run_requests: TriggerWorkflowRequest = []

        for workflow in workflows:
            workflow_name = workflow["workflow_name"]
            input_data = workflow["input"]
            options = workflow["options"]

            if namespace != "" and not workflow_name.startswith(self.namespace):
                workflow_name = f"{namespace}{workflow_name}"

            # Prepare and trigger workflow for each workflow name and input
            request = self._prepare_workflow_request(workflow_name, input_data, options)
            workflow_run_requests.append(request)

        request = BulkTriggerWorkflowRequest(workflows=workflow_run_requests)

        resp: BulkTriggerWorkflowResponse = await self.aio_client.BulkTriggerWorkflow(
            request,
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

    @tenacity_retry
    async def put_workflow(
        self,
        name: str,
        workflow: CreateWorkflowVersionOpts | WorkflowMeta,
        overrides: CreateWorkflowVersionOpts | None = None,
    ) -> WorkflowVersion:
        opts = self._prepare_put_workflow_request(name, workflow, overrides)

        return await self.aio_client.PutWorkflow(
            opts,
            metadata=get_metadata(self.token),
        )

    @tenacity_retry
    async def put_rate_limit(
        self,
        key: str,
        limit: int,
        duration: RateLimitDuration = RateLimitDuration.SECOND,
    ):
        await self.aio_client.PutRateLimit(
            PutRateLimitRequest(
                key=key,
                limit=limit,
                duration=duration,
            ),
            metadata=get_metadata(self.token),
        )

    @tenacity_retry
    async def schedule_workflow(
        self,
        name: str,
        schedules: List[Union[datetime, timestamp_pb2.Timestamp]],
        input={},
        options: ScheduleTriggerWorkflowOptions = None,
    ) -> WorkflowVersion:
        try:
            namespace = self.namespace

            if (
                options is not None
                and "namespace" in options
                and options["namespace"] is not None
            ):
                namespace = options["namespace"]
                del options["namespace"]

            if namespace != "" and not name.startswith(self.namespace):
                name = f"{namespace}{name}"

            request = self._prepare_schedule_workflow_request(
                name, schedules, input, options
            )

            return await self.aio_client.ScheduleWorkflow(
                request,
                metadata=get_metadata(self.token),
            )
        except (grpc.aio.AioRpcError, grpc.RpcError) as e:
            if e.code() == grpc.StatusCode.ALREADY_EXISTS:
                raise DedupeViolationErr(e.details())

            raise e


class AdminClient(AdminClientBase):
    def __init__(self, config: ClientConfig):
        conn = new_conn(config)
        self.config = config
        self.client = WorkflowServiceStub(conn)
        self.aio = AdminClientAioImpl(config)
        self.token = config.token
        self.listener_client = new_listener(config)
        self.namespace = config.namespace

    @tenacity_retry
    def put_workflow(
        self,
        name: str,
        workflow: CreateWorkflowVersionOpts | WorkflowMeta,
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
        duration: Union[RateLimitDuration.Value, str] = RateLimitDuration.SECOND,
    ):
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
        schedules: List[Union[datetime, timestamp_pb2.Timestamp]],
        input={},
        options: ScheduleTriggerWorkflowOptions = None,
    ) -> WorkflowVersion:
        try:
            namespace = self.namespace

            if (
                options is not None
                and "namespace" in options
                and options["namespace"] is not None
            ):
                namespace = options["namespace"]
                del options["namespace"]

            if namespace != "" and not name.startswith(self.namespace):
                name = f"{namespace}{name}"

            request = self._prepare_schedule_workflow_request(
                name, schedules, input, options
            )

            return self.client.ScheduleWorkflow(
                request,
                metadata=get_metadata(self.token),
            )
        except (grpc.RpcError, grpc.aio.AioRpcError) as e:
            if e.code() == grpc.StatusCode.ALREADY_EXISTS:
                raise DedupeViolationErr(e.details())

            raise e

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    def run_workflow(
        self, workflow_name: str, input: any, options: TriggerWorkflowOptions = None
    ) -> WorkflowRunRef:
        try:
            if not self.pooled_workflow_listener:
                self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

            namespace = self.namespace

            ## TODO: Factor this out - it's repeated a lot of places
            if (
                options is not None
                and "namespace" in options
                and options["namespace"] is not None
            ):
                namespace = options.pop("namespace")

            if namespace != "" and not workflow_name.startswith(self.namespace):
                workflow_name = f"{namespace}{workflow_name}"

            request = self._prepare_workflow_request(workflow_name, input, options)

            resp: TriggerWorkflowResponse = self.client.TriggerWorkflow(
                request,
                metadata=get_metadata(self.token),
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

    ## IMPORTANT: Keep this method's signature in sync with the wrapper in the OTel instrumentor
    @tenacity_retry
    def run_workflows(
        self, workflows: List[WorkflowRunDict], options: TriggerWorkflowOptions = None
    ) -> list[WorkflowRunRef]:
        workflow_run_requests: TriggerWorkflowRequest = []
        if not self.pooled_workflow_listener:
            self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

        for workflow in workflows:
            workflow_name = workflow["workflow_name"]
            input_data = workflow["input"]
            options = workflow["options"]

            namespace = self.namespace

            if (
                options is not None
                and "namespace" in options
                and options["namespace"] is not None
            ):
                namespace = options["namespace"]
                del options["namespace"]

            if namespace != "" and not workflow_name.startswith(self.namespace):
                workflow_name = f"{namespace}{workflow_name}"

            # Prepare and trigger workflow for each workflow name and input
            request = self._prepare_workflow_request(workflow_name, input_data, options)

            workflow_run_requests.append(request)

            request = BulkTriggerWorkflowRequest(workflows=workflow_run_requests)

        resp: BulkTriggerWorkflowResponse = self.client.BulkTriggerWorkflow(
            request,
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

    def run(
        self,
        function: Union[str, Callable[[Any], T]],
        input: any,
        options: TriggerWorkflowOptions = None,
    ) -> "RunRef[T]":
        workflow_name = function

        if not isinstance(function, str):
            workflow_name = function.function_name

        wrr = self.run_workflow(workflow_name, input, options)

        return RunRef[T](
            wrr.workflow_run_id, wrr.workflow_listener, wrr.workflow_run_event_listener
        )

    def get_workflow_run(self, workflow_run_id: str) -> WorkflowRunRef:
        try:
            if not self.pooled_workflow_listener:
                self.pooled_workflow_listener = PooledWorkflowRunListener(self.config)

            return WorkflowRunRef(
                workflow_run_id=workflow_run_id,
                workflow_listener=self.pooled_workflow_listener,
                workflow_run_event_listener=self.listener_client,
            )
        except grpc.RpcError as e:
            raise ValueError(f"Could not get workflow run: {e}")
