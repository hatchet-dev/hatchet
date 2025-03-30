from hatchet_sdk.clients.rest.api.tenant_api import TenantApi
from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.tenant_queue_metrics import TenantQueueMetrics
from hatchet_sdk.clients.rest.models.tenant_step_run_queue_metrics import (
    TenantStepRunQueueMetrics,
)
from hatchet_sdk.clients.rest.models.workflow_metrics import WorkflowMetrics
from hatchet_sdk.clients.rest.models.workflow_run_status import WorkflowRunStatus
from hatchet_sdk.clients.v1.api_client import (
    BaseRestClient,
    maybe_additional_metadata_to_kv,
)
from hatchet_sdk.utils.aio import run_async_from_sync
from hatchet_sdk.utils.typing import JSONSerializableMapping


class MetricsClient(BaseRestClient):
    def _wa(self, client: ApiClient) -> WorkflowApi:
        return WorkflowApi(client)

    def _ta(self, client: ApiClient) -> TenantApi:
        return TenantApi(client)

    async def aio_get_workflow_metrics(
        self,
        workflow_id: str,
        status: WorkflowRunStatus | None = None,
        group_key: str | None = None,
    ) -> WorkflowMetrics:
        async with self.client() as client:
            return await self._wa(client).workflow_get_metrics(
                workflow=workflow_id, status=status, group_key=group_key
            )

    def get_workflow_metrics(
        self,
        workflow_id: str,
        status: WorkflowRunStatus | None = None,
        group_key: str | None = None,
    ) -> WorkflowMetrics:
        return run_async_from_sync(
            self.aio_get_workflow_metrics, workflow_id, status, group_key
        )

    async def aio_get_queue_metrics(
        self,
        workflow_ids: list[str] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
    ) -> TenantQueueMetrics:
        async with self.client() as client:
            return await self._wa(client).tenant_get_queue_metrics(
                tenant=self.client_config.tenant_id,
                workflows=workflow_ids,
                additional_metadata=maybe_additional_metadata_to_kv(
                    additional_metadata
                ),
            )

    def get_queue_metrics(
        self,
        workflow_ids: list[str] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
    ) -> TenantQueueMetrics:
        return run_async_from_sync(
            self.aio_get_queue_metrics, workflow_ids, additional_metadata
        )

    async def aio_get_task_metrics(self) -> TenantStepRunQueueMetrics:
        async with self.client() as client:
            return await self._ta(client).tenant_get_step_run_queue_metrics(
                tenant=self.client_config.tenant_id
            )

    def get_task_metrics(self) -> TenantStepRunQueueMetrics:
        return run_async_from_sync(self.aio_get_task_metrics)
