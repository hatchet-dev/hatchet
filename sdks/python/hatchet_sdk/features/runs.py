import asyncio
from datetime import datetime, timedelta
from typing import TYPE_CHECKING, Literal, overload

from pydantic import BaseModel, model_validator

from hatchet_sdk.clients.listeners.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.listeners.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.clients.rest.api.task_api import TaskApi
from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_cancel_task_request import V1CancelTaskRequest
from hatchet_sdk.clients.rest.models.v1_replay_task_request import V1ReplayTaskRequest
from hatchet_sdk.clients.rest.models.v1_task_filter import V1TaskFilter
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary_list import V1TaskSummaryList
from hatchet_sdk.clients.rest.models.v1_trigger_workflow_run_request import (
    V1TriggerWorkflowRunRequest,
)
from hatchet_sdk.clients.rest.models.v1_workflow_run_details import V1WorkflowRunDetails
from hatchet_sdk.clients.v1.api_client import (
    BaseRestClient,
    maybe_additional_metadata_to_kv,
)
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.utils.typing import JSONSerializableMapping

if TYPE_CHECKING:
    from hatchet_sdk.workflow_run import WorkflowRunRef


class RunFilter(BaseModel):
    since: datetime
    until: datetime | None = None
    statuses: list[V1TaskStatus] | None = None
    workflow_ids: list[str] | None = None
    additional_metadata: dict[str, str] | None = None


class BulkCancelReplayOpts(BaseModel):
    ids: list[str] | None = None
    filters: RunFilter | None = None

    @model_validator(mode="after")
    def validate_model(self) -> "BulkCancelReplayOpts":
        if not self.ids and not self.filters:
            raise ValueError("ids or filters must be set")

        if self.ids and self.filters:
            raise ValueError("ids and filters cannot both be set")

        return self

    @property
    def v1_task_filter(self) -> V1TaskFilter | None:
        if not self.filters:
            return None

        return V1TaskFilter(
            since=self.filters.since,
            until=self.filters.until,
            statuses=self.filters.statuses,
            workflowIds=self.filters.workflow_ids,
            additionalMetadata=maybe_additional_metadata_to_kv(
                self.filters.additional_metadata
            ),
        )

    @overload
    def to_request(self, request_type: Literal["replay"]) -> V1ReplayTaskRequest: ...

    @overload
    def to_request(self, request_type: Literal["cancel"]) -> V1CancelTaskRequest: ...

    def to_request(
        self, request_type: Literal["replay", "cancel"]
    ) -> V1ReplayTaskRequest | V1CancelTaskRequest:
        if request_type == "replay":
            return V1ReplayTaskRequest(
                externalIds=self.ids,
                filter=self.v1_task_filter,
            )

        if request_type == "cancel":
            return V1CancelTaskRequest(
                externalIds=self.ids,
                filter=self.v1_task_filter,
            )


class RunsClient(BaseRestClient):
    def __init__(
        self,
        config: ClientConfig,
        workflow_run_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
    ) -> None:
        super().__init__(config)

        self.workflow_run_listener = workflow_run_listener
        self.workflow_run_event_listener = workflow_run_event_listener

    def _wra(self, client: ApiClient) -> WorkflowRunsApi:
        return WorkflowRunsApi(client)

    def _ta(self, client: ApiClient) -> TaskApi:
        return TaskApi(client)

    def get(self, workflow_run_id: str) -> V1WorkflowRunDetails:
        with self.client() as client:
            return self._wra(client).v1_workflow_run_get(str(workflow_run_id))

    async def aio_get(self, workflow_run_id: str) -> V1WorkflowRunDetails:
        return await asyncio.to_thread(self.get, workflow_run_id)

    async def aio_list(
        self,
        since: datetime = datetime.now() - timedelta(hours=1),
        only_tasks: bool = False,
        offset: int | None = None,
        limit: int | None = None,
        statuses: list[V1TaskStatus] | None = None,
        until: datetime | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
        worker_id: str | None = None,
        parent_task_external_id: str | None = None,
    ) -> V1TaskSummaryList:
        return await asyncio.to_thread(
            self.list,
            since=since,
            only_tasks=only_tasks,
            offset=offset,
            limit=limit,
            statuses=statuses,
            until=until,
            additional_metadata=additional_metadata,
            workflow_ids=workflow_ids,
            worker_id=worker_id,
            parent_task_external_id=parent_task_external_id,
        )

    def list(
        self,
        since: datetime = datetime.now() - timedelta(hours=1),
        only_tasks: bool = False,
        offset: int | None = None,
        limit: int | None = None,
        statuses: list[V1TaskStatus] | None = None,
        until: datetime | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
        worker_id: str | None = None,
        parent_task_external_id: str | None = None,
    ) -> V1TaskSummaryList:
        with self.client() as client:
            return self._wra(client).v1_workflow_run_list(
                tenant=self.client_config.tenant_id,
                since=since,
                only_tasks=only_tasks,
                offset=offset,
                limit=limit,
                statuses=statuses,
                until=until,
                additional_metadata=maybe_additional_metadata_to_kv(
                    additional_metadata
                ),
                workflow_ids=workflow_ids,
                worker_id=worker_id,
                parent_task_external_id=parent_task_external_id,
            )

    def create(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping = {},
        priority: int | None = None,
    ) -> V1WorkflowRunDetails:
        with self.client() as client:
            return self._wra(client).v1_workflow_run_create(
                tenant=self.client_config.tenant_id,
                v1_trigger_workflow_run_request=V1TriggerWorkflowRunRequest(
                    workflowName=workflow_name,
                    input=dict(input),
                    additionalMetadata=dict(additional_metadata),
                    priority=priority,
                ),
            )

    async def aio_create(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping = {},
        priority: int | None = None,
    ) -> V1WorkflowRunDetails:
        return await asyncio.to_thread(
            self.create, workflow_name, input, additional_metadata, priority
        )

    def replay(self, run_id: str) -> None:
        self.bulk_replay(opts=BulkCancelReplayOpts(ids=[run_id]))

    async def aio_replay(self, run_id: str) -> None:
        return await asyncio.to_thread(self.replay, run_id)

    def bulk_replay(self, opts: BulkCancelReplayOpts) -> None:
        with self.client() as client:
            self._ta(client).v1_task_replay(
                tenant=self.client_config.tenant_id,
                v1_replay_task_request=opts.to_request("replay"),
            )

    async def aio_bulk_replay(self, opts: BulkCancelReplayOpts) -> None:
        return await asyncio.to_thread(self.bulk_replay, opts)

    def cancel(self, run_id: str) -> None:
        self.bulk_cancel(opts=BulkCancelReplayOpts(ids=[run_id]))

    async def aio_cancel(self, run_id: str) -> None:
        return await asyncio.to_thread(self.cancel, run_id)

    def bulk_cancel(self, opts: BulkCancelReplayOpts) -> None:
        with self.client() as client:
            self._ta(client).v1_task_cancel(
                tenant=self.client_config.tenant_id,
                v1_cancel_task_request=opts.to_request("cancel"),
            )

    async def aio_bulk_cancel(self, opts: BulkCancelReplayOpts) -> None:
        return await asyncio.to_thread(self.bulk_cancel, opts)

    def get_result(self, run_id: str) -> JSONSerializableMapping:
        details = self.get(run_id)

        return details.run.output

    def get_run_ref(self, workflow_run_id: str) -> "WorkflowRunRef":
        from hatchet_sdk.workflow_run import WorkflowRunRef

        return WorkflowRunRef(
            workflow_run_id=workflow_run_id,
            workflow_run_event_listener=self.workflow_run_event_listener,
            workflow_run_listener=self.workflow_run_listener,
            runs_client=self,
        )

    async def aio_get_result(self, run_id: str) -> JSONSerializableMapping:
        details = await asyncio.to_thread(self.get, run_id)

        return details.run.output
