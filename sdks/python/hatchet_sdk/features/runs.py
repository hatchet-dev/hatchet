from datetime import datetime, timedelta
from uuid import UUID

from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary_list import V1TaskSummaryList
from hatchet_sdk.clients.rest.models.v1_workflow_run_details import V1WorkflowRunDetails
from hatchet_sdk.clients.v1.api_client import BaseRestClient
from hatchet_sdk.config import ClientConfig


class RunsClient(BaseRestClient):
    def __init__(self, config: ClientConfig) -> None:
        super().__init__(config)

    def _wra(self, client: ApiClient) -> WorkflowRunsApi:
        return WorkflowRunsApi(client)

    async def aio_get(self, workflow_run_id: UUID) -> V1WorkflowRunDetails:
        async with self.client() as client:
            return await self._wra(client).v1_workflow_run_get(str(workflow_run_id))

    def get(self, workflow_run_id: UUID) -> V1WorkflowRunDetails:
        return self._run_async_from_sync(self.aio_get, workflow_run_id)

    async def aio_list(
        self,
        since: datetime = datetime.now() - timedelta(hours=1),
        only_tasks: bool = False,
        offset: int | None = None,
        limit: int | None = None,
        statuses: list[V1TaskStatus] | None = None,
        until: datetime | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[UUID] | None = None,
        worker_id: UUID | None = None,
        parent_task_external_id: UUID | None = None,
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
                additional_metadata=self.maybe_additional_metadata_to_kv(
                    additional_metadata
                ),
                workflow_ids=self.maybe_uuid_list_to_str_list(workflow_ids),
                worker_id=self.maybe_uuid_to_str(worker_id),
                parent_task_external_id=self.maybe_uuid_to_str(parent_task_external_id),
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
        workflow_ids: list[UUID] | None = None,
        worker_id: UUID | None = None,
        parent_task_external_id: UUID | None = None,
    ) -> V1TaskSummaryList:
        return self._run_async_from_sync(
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
