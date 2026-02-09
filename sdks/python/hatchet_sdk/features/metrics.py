import asyncio
from datetime import datetime, timedelta, timezone
from typing import Any

from pydantic import BaseModel

from hatchet_sdk.clients.rest.api.task_api import TaskApi
from hatchet_sdk.clients.rest.api.tenant_api import TenantApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.task_stat import TaskStat
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import BaseRestClient


class TaskMetrics(BaseModel):
    cancelled: int
    completed: int
    failed: int
    queued: int
    running: int


class MetricsClient(BaseRestClient):
    """
    The metrics client is a client for reading metrics out of Hatchet's metrics API.
    """

    def _taskapi(self, client: ApiClient) -> TaskApi:
        return TaskApi(client)

    def _ta(self, client: ApiClient) -> TenantApi:
        return TenantApi(client)

    def get_queue_metrics(
        self,
    ) -> dict[str, Any]:
        """
        Retrieve the current queue metrics for the tenant.

        :return: The current queue metrics
        """
        with self.client() as client:
            tenant_get_step_run_queue_metrics = tenacity_retry(
                self._ta(client).tenant_get_step_run_queue_metrics,
                self.client_config.tenacity,
            )
            return (
                tenant_get_step_run_queue_metrics(
                    tenant=self.client_config.tenant_id,
                ).queues
            ) or {}

    async def aio_get_queue_metrics(
        self,
    ) -> dict[str, Any]:
        """
        Retrieve the current queue metrics for the tenant.

        :return: The current queue metrics
        """

        return await asyncio.to_thread(self.get_queue_metrics)

    def scrape_tenant_prometheus_metrics(
        self,
    ) -> str:
        """
        Scrape Prometheus metrics for the tenant. Returns the metrics in Prometheus text format.

        :return: The metrics, returned in Prometheus text format
        """
        with self.client() as client:
            tenant_get_prometheus_metrics = tenacity_retry(
                self._ta(client).tenant_get_prometheus_metrics,
                self.client_config.tenacity,
            )
            return tenant_get_prometheus_metrics(
                tenant=self.client_config.tenant_id,
            )

    async def aio_scrape_tenant_prometheus_metrics(
        self,
    ) -> str:
        """
        Scrape Prometheus metrics for the tenant. Returns the metrics in Prometheus text format.

        :return: The metrics, returned in Prometheus text format
        """

        return await asyncio.to_thread(self.scrape_tenant_prometheus_metrics)

    def get_task_stats(self) -> dict[str, TaskStat]:
        with self.client() as client:
            get_task_stats = tenacity_retry(
                self._ta(client).tenant_get_task_stats, self.client_config.tenacity
            )
            return get_task_stats(
                tenant=self.client_config.tenant_id,
            )

    async def aio_get_task_stats(self) -> dict[str, TaskStat]:
        return await asyncio.to_thread(self.get_task_stats)

    def get_task_metrics(
        self,
        since: datetime | None = None,
        until: datetime | None = None,
        workflow_ids: list[str] | None = None,
        parent_task_external_id: str | None = None,
        triggering_event_external_id: str | None = None,
    ) -> TaskMetrics:
        """
        Retrieve task metrics, grouped by status (queued, running, completed, failed, cancelled).

        :param since: Start time for the metrics query (defaults to the past day if unset)
        :param until: End time for the metrics query
        :param workflow_ids: List of workflow IDs to filter the metrics by
        :param parent_task_external_id: ID of the parent task to filter by (note that parent task here refers to the task that spawned this task as a child)
        :param triggering_event_external_id: ID of the triggering event to filter by
        :return: Task metrics
        """

        since = since or datetime.now(timezone.utc) - timedelta(days=1)
        until = until or datetime.now(timezone.utc)
        with self.client() as client:
            v1_task_list_status_metrics = tenacity_retry(
                self._taskapi(client).v1_task_list_status_metrics,
                self.client_config.tenacity,
            )
            metrics = {
                m.status.name.lower(): m.count
                for m in v1_task_list_status_metrics(
                    tenant=self.client_config.tenant_id,
                    since=since,
                    until=until,
                    workflow_ids=workflow_ids,
                    parent_task_external_id=parent_task_external_id,
                    triggering_event_external_id=triggering_event_external_id,
                )
            }

            return TaskMetrics.model_validate(metrics)

    async def aio_get_task_metrics(
        self,
        since: datetime | None = None,
        until: datetime | None = None,
        workflow_ids: list[str] | None = None,
        parent_task_external_id: str | None = None,
        triggering_event_external_id: str | None = None,
    ) -> TaskMetrics:
        """
        Retrieve task metrics, grouped by status (queued, running, completed, failed, cancelled).

        :param since: Start time for the metrics query (defaults to the past day if unset)
        :param until: End time for the metrics query
        :param workflow_ids: List of workflow IDs to filter the metrics by
        :param parent_task_external_id: ID of the parent task to filter by (note that parent task here refers to the task that spawned this task as a child)
        :param triggering_event_external_id: ID of the triggering event to filter by
        :return: Task metrics
        """
        return await asyncio.to_thread(
            self.get_task_metrics,
            since,
            until,
            workflow_ids,
            parent_task_external_id,
            triggering_event_external_id,
        )
