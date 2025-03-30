import datetime
from typing import Optional

from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi
from hatchet_sdk.clients.rest.api.workflow_run_api import WorkflowRunApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.schedule_workflow_run_request import (
    ScheduleWorkflowRunRequest,
)
from hatchet_sdk.clients.rest.models.scheduled_run_status import ScheduledRunStatus
from hatchet_sdk.clients.rest.models.scheduled_workflows import ScheduledWorkflows
from hatchet_sdk.clients.rest.models.scheduled_workflows_list import (
    ScheduledWorkflowsList,
)
from hatchet_sdk.clients.rest.models.scheduled_workflows_order_by_field import (
    ScheduledWorkflowsOrderByField,
)
from hatchet_sdk.clients.rest.models.workflow_run_order_by_direction import (
    WorkflowRunOrderByDirection,
)
from hatchet_sdk.clients.v1.api_client import (
    BaseRestClient,
    maybe_additional_metadata_to_kv,
)
from hatchet_sdk.utils.aio import run_async_from_sync
from hatchet_sdk.utils.typing import JSONSerializableMapping


class ScheduledClient(BaseRestClient):
    def _wra(self, client: ApiClient) -> WorkflowRunApi:
        return WorkflowRunApi(client)

    def _wa(self, client: ApiClient) -> WorkflowApi:
        return WorkflowApi(client)

    async def aio_create(
        self,
        workflow_name: str,
        trigger_at: datetime.datetime,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping,
    ) -> ScheduledWorkflows:
        """
        Creates a new scheduled workflow run asynchronously.

        Args:
            workflow_name (str): The name of the scheduled workflow.
            trigger_at (datetime.datetime): The datetime when the run should be triggered.
            input (JSONSerializableMapping): The input data for the scheduled workflow.
            additional_metadata (JSONSerializableMapping): Additional metadata associated with the future run.

        Returns:
            ScheduledWorkflows: The created scheduled workflow instance.
        """
        async with self.client() as client:
            return await self._wra(client).scheduled_workflow_run_create(
                tenant=self.client_config.tenant_id,
                workflow=workflow_name,
                schedule_workflow_run_request=ScheduleWorkflowRunRequest(
                    triggerAt=trigger_at,
                    input=dict(input),
                    additionalMetadata=dict(additional_metadata),
                ),
            )

    def create(
        self,
        workflow_name: str,
        trigger_at: datetime.datetime,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping,
    ) -> ScheduledWorkflows:
        """
        Creates a new scheduled workflow run asynchronously.

        Args:
            workflow_name (str): The name of the scheduled workflow.
            trigger_at (datetime.datetime): The datetime when the run should be triggered.
            input (JSONSerializableMapping): The input data for the scheduled workflow.
            additional_metadata (JSONSerializableMapping): Additional metadata associated with the future run as a key-value pair (e.g. {"key1": "value1", "key2": "value2"}).

        Returns:
            ScheduledWorkflows: The created scheduled workflow instance.
        """

        return run_async_from_sync(
            self.aio_create,
            workflow_name,
            trigger_at,
            input,
            additional_metadata,
        )

    async def aio_delete(self, scheduled_id: str) -> None:
        """
        Deletes a scheduled workflow run.

        Args:
            scheduled_id (str): The scheduled workflow trigger ID to delete.
        """
        async with self.client() as client:
            await self._wa(client).workflow_scheduled_delete(
                tenant=self.client_config.tenant_id,
                scheduled_workflow_run=scheduled_id,
            )

    def delete(self, scheduled_id: str) -> None:
        run_async_from_sync(self.aio_delete, scheduled_id)

    async def aio_list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        parent_workflow_run_id: str | None = None,
        statuses: list[ScheduledRunStatus] | None = None,
        additional_metadata: Optional[JSONSerializableMapping] = None,
        order_by_field: Optional[ScheduledWorkflowsOrderByField] = None,
        order_by_direction: Optional[WorkflowRunOrderByDirection] = None,
    ) -> ScheduledWorkflowsList:
        """
        Retrieves a list of scheduled workflows based on provided filters.

        Args:
            offset (int | None): The starting point for the list.
            limit (int | None): The maximum number of items to return.
            workflow_id (str | None): Filter by specific workflow ID.
            parent_workflow_run_id (str | None): Filter by parent workflow run ID.
            statuses (list[ScheduledRunStatus] | None): Filter by status.
            additional_metadata (Optional[List[dict[str, str]]]): Filter by additional metadata.
            order_by_field (Optional[ScheduledWorkflowsOrderByField]): Field to order the results by.
            order_by_direction (Optional[WorkflowRunOrderByDirection]): Direction to order the results.

        Returns:
            List[ScheduledWorkflows]: A list of scheduled workflows matching the criteria.
        """
        async with self.client() as client:
            return await self._wa(client).workflow_scheduled_list(
                tenant=self.client_config.tenant_id,
                offset=offset,
                limit=limit,
                order_by_field=order_by_field,
                order_by_direction=order_by_direction,
                workflow_id=workflow_id,
                additional_metadata=maybe_additional_metadata_to_kv(
                    additional_metadata
                ),
                parent_workflow_run_id=parent_workflow_run_id,
                statuses=statuses,
            )

    def list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        parent_workflow_run_id: str | None = None,
        statuses: list[ScheduledRunStatus] | None = None,
        additional_metadata: Optional[JSONSerializableMapping] = None,
        order_by_field: Optional[ScheduledWorkflowsOrderByField] = None,
        order_by_direction: Optional[WorkflowRunOrderByDirection] = None,
    ) -> ScheduledWorkflowsList:
        """
        Retrieves a list of scheduled workflows based on provided filters.

        Args:
            offset (int | None): The starting point for the list.
            limit (int | None): The maximum number of items to return.
            workflow_id (str | None): Filter by specific workflow ID.
            parent_workflow_run_id (str | None): Filter by parent workflow run ID.
            statuses (list[ScheduledRunStatus] | None): Filter by status.
            additional_metadata (Optional[List[dict[str, str]]]): Filter by additional metadata.
            order_by_field (Optional[ScheduledWorkflowsOrderByField]): Field to order the results by.
            order_by_direction (Optional[WorkflowRunOrderByDirection]): Direction to order the results.

        Returns:
            List[ScheduledWorkflows]: A list of scheduled workflows matching the criteria.
        """
        return run_async_from_sync(
            self.aio_list,
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
            parent_workflow_run_id=parent_workflow_run_id,
            statuses=statuses,
        )

    async def aio_get(self, scheduled_id: str) -> ScheduledWorkflows:
        """
        Retrieves a specific scheduled workflow by scheduled run trigger ID.

        Args:
            scheduled (str): The scheduled workflow trigger ID to retrieve.

        Returns:
            ScheduledWorkflows: The requested scheduled workflow instance.
        """

        async with self.client() as client:
            return await self._wa(client).workflow_scheduled_get(
                tenant=self.client_config.tenant_id,
                scheduled_workflow_run=scheduled_id,
            )

    def get(self, scheduled_id: str) -> ScheduledWorkflows:
        """
        Retrieves a specific scheduled workflow by scheduled run trigger ID.

        Args:
            scheduled (str): The scheduled workflow trigger ID to retrieve.

        Returns:
            ScheduledWorkflows: The requested scheduled workflow instance.
        """
        return run_async_from_sync(self.aio_get, scheduled_id)
