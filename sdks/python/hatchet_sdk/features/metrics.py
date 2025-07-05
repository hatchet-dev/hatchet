import asyncio

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
    retry,
)
from hatchet_sdk.utils.typing import JSONSerializableMapping


class MetricsClient(BaseRestClient):
    """
    The metrics client is a client for reading metrics out of Hatchet's metrics API.
    """

    def _wa(self, client: ApiClient) -> WorkflowApi:
        return WorkflowApi(client)

    def _ta(self, client: ApiClient) -> TenantApi:
        return TenantApi(client)

    @retry
    def get_workflow_metrics(
        self,
        workflow_id: str,
        status: WorkflowRunStatus | None = None,
        group_key: str | None = None,
    ) -> WorkflowMetrics:
        """
        Retrieve workflow metrics for a given workflow ID.

        :param workflow_id: The ID of the workflow to retrieve metrics for.
        :param status: The status of the workflow run to filter by.
        :param group_key: The key to group the metrics by.

        :return: Workflow metrics for the specified workflow ID.
        """
        with self.client() as client:
            return self._wa(client).workflow_get_metrics(
                workflow=workflow_id, status=status, group_key=group_key
            )

    async def aio_get_workflow_metrics(
        self,
        workflow_id: str,
        status: WorkflowRunStatus | None = None,
        group_key: str | None = None,
    ) -> WorkflowMetrics:
        """
        Retrieve workflow metrics for a given workflow ID.

        :param workflow_id: The ID of the workflow to retrieve metrics for.
        :param status: The status of the workflow run to filter by.
        :param group_key: The key to group the metrics by.

        :return: Workflow metrics for the specified workflow ID.
        """
        return await asyncio.to_thread(
            self.get_workflow_metrics, workflow_id, status, group_key
        )

    @retry
    def get_queue_metrics(
        self,
        workflow_ids: list[str] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
    ) -> TenantQueueMetrics:
        """
        Retrieve queue metrics for a set of workflow ids and additional metadata.

        :param workflow_ids: A list of workflow IDs to retrieve metrics for.
        :param additional_metadata: Additional metadata to filter the metrics by.

        :return: Workflow metrics for the specified workflow IDs.
        """
        with self.client() as client:
            return self._wa(client).tenant_get_queue_metrics(
                tenant=self.client_config.tenant_id,
                workflows=workflow_ids,
                additional_metadata=maybe_additional_metadata_to_kv(
                    additional_metadata
                ),
            )

    async def aio_get_queue_metrics(
        self,
        workflow_ids: list[str] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
    ) -> TenantQueueMetrics:
        """
        Retrieve queue metrics for a set of workflow ids and additional metadata.

        :param workflow_ids: A list of workflow IDs to retrieve metrics for.
        :param additional_metadata: Additional metadata to filter the metrics by.

        :return: Workflow metrics for the specified workflow IDs.
        """
        return await asyncio.to_thread(
            self.get_queue_metrics, workflow_ids, additional_metadata
        )

    @retry
    def get_task_metrics(self) -> TenantStepRunQueueMetrics:
        """
        Retrieve queue metrics

        :return: Step run queue metrics for the tenant
        """
        with self.client() as client:
            return self._ta(client).tenant_get_step_run_queue_metrics(
                tenant=self.client_config.tenant_id
            )

    async def aio_get_task_metrics(self) -> TenantStepRunQueueMetrics:
        """
        Retrieve queue metrics

        :return: Step run queue metrics for the tenant
        """
        return await asyncio.to_thread(self.get_task_metrics)
