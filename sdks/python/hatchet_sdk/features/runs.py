from datetime import datetime, timedelta
from typing import Literal, overload

from pydantic import BaseModel, model_validator

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
from hatchet_sdk.utils.aio import run_async_from_sync
from hatchet_sdk.utils.typing import JSONSerializableMapping
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
    def _wra(self, client: ApiClient) -> WorkflowRunsApi:
        return WorkflowRunsApi(client)

    def _ta(self, client: ApiClient) -> TaskApi:
        return TaskApi(client)

    async def aio_get(self, workflow_run_id: str) -> V1WorkflowRunDetails:
        async with self.client() as client:
            return await self._wra(client).v1_workflow_run_get(str(workflow_run_id))

    def get(self, workflow_run_id: str) -> V1WorkflowRunDetails:
        return run_async_from_sync(self.aio_get, workflow_run_id)

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
        async with self.client() as client:
            return await self._wra(client).v1_workflow_run_list(
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
        return run_async_from_sync(
            self.aio_list,
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

    async def aio_create(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping = {},
    ) -> V1WorkflowRunDetails:
        async with self.client() as client:
            return await self._wra(client).v1_workflow_run_create(
                tenant=self.client_config.tenant_id,
                v1_trigger_workflow_run_request=V1TriggerWorkflowRunRequest(
                    workflowName=workflow_name,
                    input=dict(input),
                    additionalMetadata=dict(additional_metadata),
                ),
            )

    def create(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping = {},
    ) -> V1WorkflowRunDetails:
        return run_async_from_sync(
            self.aio_create, workflow_name, input, additional_metadata
        )

    async def aio_replay(self, run_id: str) -> None:
        await self.aio_bulk_replay(opts=BulkCancelReplayOpts(ids=[run_id]))

    def replay(self, run_id: str) -> None:
        return run_async_from_sync(self.aio_replay, run_id)

    async def aio_bulk_replay(self, opts: BulkCancelReplayOpts) -> None:
        async with self.client() as client:
            await self._ta(client).v1_task_replay(
                tenant=self.client_config.tenant_id,
                v1_replay_task_request=opts.to_request("replay"),
            )

    def bulk_replay(self, opts: BulkCancelReplayOpts) -> None:
        return run_async_from_sync(self.aio_bulk_replay, opts)

    async def aio_cancel(self, run_id: str) -> None:
        await self.aio_bulk_cancel(opts=BulkCancelReplayOpts(ids=[run_id]))

    def cancel(self, run_id: str) -> None:
        return run_async_from_sync(self.aio_cancel, run_id)

    async def aio_bulk_cancel(self, opts: BulkCancelReplayOpts) -> None:
        async with self.client() as client:
            await self._ta(client).v1_task_cancel(
                tenant=self.client_config.tenant_id,
                v1_cancel_task_request=opts.to_request("cancel"),
            )

    def bulk_cancel(self, opts: BulkCancelReplayOpts) -> None:
        return run_async_from_sync(self.aio_bulk_cancel, opts)

    async def aio_get_result(self, run_id: str) -> JSONSerializableMapping:
        details = await self.aio_get(run_id)

        return details.run.output

    def get_result(self, run_id: str) -> JSONSerializableMapping:
        details = self.get(run_id)

        return details.run.output

    def get_run_ref(self, workflow_run_id: str) -> WorkflowRunRef:
        return WorkflowRunRef(
            workflow_run_id=workflow_run_id,
            config=self.client_config,
        )
