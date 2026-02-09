import asyncio
import datetime

from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi
from hatchet_sdk.clients.rest.api.workflow_run_api import WorkflowRunApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.schedule_workflow_run_request import (
    ScheduleWorkflowRunRequest,
)
from hatchet_sdk.clients.rest.models.scheduled_run_status import ScheduledRunStatus
from hatchet_sdk.clients.rest.models.scheduled_workflows import ScheduledWorkflows
from hatchet_sdk.clients.rest.models.scheduled_workflows_bulk_delete_filter import (
    ScheduledWorkflowsBulkDeleteFilter,
)
from hatchet_sdk.clients.rest.models.scheduled_workflows_bulk_delete_request import (
    ScheduledWorkflowsBulkDeleteRequest,
)
from hatchet_sdk.clients.rest.models.scheduled_workflows_bulk_delete_response import (
    ScheduledWorkflowsBulkDeleteResponse,
)
from hatchet_sdk.clients.rest.models.scheduled_workflows_bulk_update_item import (
    ScheduledWorkflowsBulkUpdateItem,
)
from hatchet_sdk.clients.rest.models.scheduled_workflows_bulk_update_request import (
    ScheduledWorkflowsBulkUpdateRequest,
)
from hatchet_sdk.clients.rest.models.scheduled_workflows_bulk_update_response import (
    ScheduledWorkflowsBulkUpdateResponse,
)
from hatchet_sdk.clients.rest.models.scheduled_workflows_list import (
    ScheduledWorkflowsList,
)
from hatchet_sdk.clients.rest.models.scheduled_workflows_order_by_field import (
    ScheduledWorkflowsOrderByField,
)
from hatchet_sdk.clients.rest.models.update_scheduled_workflow_run_request import (
    UpdateScheduledWorkflowRunRequest,
)
from hatchet_sdk.clients.rest.models.workflow_run_order_by_direction import (
    WorkflowRunOrderByDirection,
)
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import (
    BaseRestClient,
    maybe_additional_metadata_to_kv,
)
from hatchet_sdk.utils.typing import JSONSerializableMapping


class ScheduledClient(BaseRestClient):
    """
    The scheduled client is a client for managing scheduled workflows within Hatchet.
    """

    def _wra(self, client: ApiClient) -> WorkflowRunApi:
        return WorkflowRunApi(client)

    def _wa(self, client: ApiClient) -> WorkflowApi:
        return WorkflowApi(client)

    def create(
        self,
        workflow_name: str,
        trigger_at: datetime.datetime,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping,
    ) -> ScheduledWorkflows:
        """
        Creates a new scheduled workflow run.

        IMPORTANT: It's preferable to use `Workflow.run` (and similar) to trigger workflows if possible. This method is intended to be an escape hatch. For more details, see [the documentation](https://docs.hatchet.run/sdks/python/runnables#workflow).

        :param workflow_name: The name of the workflow to schedule.
        :param trigger_at: The datetime when the run should be triggered.
        :param input: The input data for the scheduled workflow.
        :param additional_metadata: Additional metadata associated with the future run as a key-value pair.

        :return: The created scheduled workflow instance.
        """
        with self.client() as client:
            return self._wra(client).scheduled_workflow_run_create(
                tenant=self.client_config.tenant_id,
                workflow=self.client_config.apply_namespace(workflow_name),
                schedule_workflow_run_request=ScheduleWorkflowRunRequest(
                    triggerAt=trigger_at,
                    input=input,
                    additionalMetadata=additional_metadata,
                ),
            )

    async def aio_create(
        self,
        workflow_name: str,
        trigger_at: datetime.datetime,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping,
    ) -> ScheduledWorkflows:
        """
        Creates a new scheduled workflow run.

        IMPORTANT: It's preferable to use `Workflow.run` (and similar) to trigger workflows if possible. This method is intended to be an escape hatch. For more details, see [the documentation](https://docs.hatchet.run/sdks/python/runnables#workflow).

        :param workflow_name: The name of the workflow to schedule.
        :param trigger_at: The datetime when the run should be triggered.
        :param input: The input data for the scheduled workflow.
        :param additional_metadata: Additional metadata associated with the future run as a key-value pair.

        :return: The created scheduled workflow instance.
        """

        return await asyncio.to_thread(
            self.create,
            workflow_name,
            trigger_at,
            input,
            additional_metadata,
        )

    def delete(self, scheduled_id: str) -> None:
        """
        Deletes a scheduled workflow run by its ID.

        :param scheduled_id: The ID of the scheduled workflow run to delete.
        :return: None
        """
        with self.client() as client:
            self._wa(client).workflow_scheduled_delete(
                tenant=self.client_config.tenant_id,
                scheduled_workflow_run=scheduled_id,
            )

    async def aio_delete(self, scheduled_id: str) -> None:
        """
        Deletes a scheduled workflow run by its ID.

        :param scheduled_id: The ID of the scheduled workflow run to delete.
        :return: None
        """
        await asyncio.to_thread(self.delete, scheduled_id)

    def update(
        self,
        scheduled_id: str,
        trigger_at: datetime.datetime,
    ) -> ScheduledWorkflows:
        """
        Reschedule a scheduled workflow run by its ID.

        Note: the server may reject rescheduling if the scheduled run has already
        triggered, or if it was created via code definition (not via API).

        :param scheduled_id: The ID of the scheduled workflow run to reschedule.
        :param trigger_at: The datetime when the run should be triggered.
        :return: The updated scheduled workflow instance.
        """
        with self.client() as client:
            return self._wa(client).workflow_scheduled_update(
                tenant=self.client_config.tenant_id,
                scheduled_workflow_run=scheduled_id,
                update_scheduled_workflow_run_request=UpdateScheduledWorkflowRunRequest(
                    triggerAt=trigger_at
                ),
            )

    async def aio_update(
        self,
        scheduled_id: str,
        trigger_at: datetime.datetime,
    ) -> ScheduledWorkflows:
        """
        Reschedule a scheduled workflow run by its ID.

        :param scheduled_id: The ID of the scheduled workflow run to reschedule.
        :param trigger_at: The datetime when the run should be triggered.
        :return: The updated scheduled workflow instance.
        """
        return await asyncio.to_thread(self.update, scheduled_id, trigger_at)

    def bulk_delete(
        self,
        *,
        scheduled_ids: list[str] | None = None,
        workflow_id: str | None = None,
        parent_workflow_run_id: str | None = None,
        parent_step_run_id: str | None = None,
        statuses: list[ScheduledRunStatus] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
    ) -> ScheduledWorkflowsBulkDeleteResponse:
        """
        Bulk delete scheduled workflow runs.

        Provide either:
        - `scheduled_ids` (explicit list of scheduled run IDs), or
        - one or more filter fields (`workflow_id`, `parent_workflow_run_id`, `parent_step_run_id`,
          `statuses`, `additional_metadata`)

        :param scheduled_ids: Explicit list of scheduled workflow run IDs to delete.
        :param workflow_id: Filter by workflow ID.
        :param parent_workflow_run_id: Filter by parent workflow run ID.
        :param parent_step_run_id: Filter by parent step run ID.
        :param statuses: Filter by scheduled run statuses.
        :param additional_metadata: Filter by additional metadata key/value pairs.
        :return: The bulk delete response containing deleted IDs and per-item errors.
        :raises ValueError: If neither `scheduled_ids` nor any filter field is provided.
        """

        if statuses:
            self.client_config.logger.warning(
                "The 'statuses' filter is not supported for bulk delete and will be ignored."
            )

        has_filter = any(
            v is not None
            for v in (
                workflow_id,
                parent_workflow_run_id,
                parent_step_run_id,
                additional_metadata,
            )
        )

        if not scheduled_ids and not has_filter:
            raise ValueError(
                "bulk_delete requires either scheduled_ids or at least one filter field."
            )

        filter_obj = None
        if has_filter:
            filter_obj = ScheduledWorkflowsBulkDeleteFilter(
                workflowId=workflow_id,
                parentWorkflowRunId=parent_workflow_run_id,
                parentStepRunId=parent_step_run_id,
                additionalMetadata=maybe_additional_metadata_to_kv(additional_metadata),
            )

        with self.client() as client:
            return self._wa(client).workflow_scheduled_bulk_delete(
                tenant=self.client_config.tenant_id,
                scheduled_workflows_bulk_delete_request=ScheduledWorkflowsBulkDeleteRequest(
                    scheduledWorkflowRunIds=scheduled_ids,
                    filter=filter_obj,
                ),
            )

    async def aio_bulk_delete(
        self,
        *,
        scheduled_ids: list[str] | None = None,
        workflow_id: str | None = None,
        parent_workflow_run_id: str | None = None,
        parent_step_run_id: str | None = None,
        statuses: list[ScheduledRunStatus] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
    ) -> ScheduledWorkflowsBulkDeleteResponse:
        """
        Bulk delete scheduled workflow runs.

        :param scheduled_ids: Explicit list of scheduled workflow run IDs to delete.
        :param workflow_id: Filter by workflow ID.
        :param parent_workflow_run_id: Filter by parent workflow run ID.
        :param parent_step_run_id: Filter by parent step run ID.
        :param statuses: Filter by scheduled run statuses.
        :param additional_metadata: Filter by additional metadata key/value pairs.
        :return: The bulk delete response containing deleted IDs and per-item errors.
        :raises ValueError: If neither `scheduled_ids` nor any filter field is provided.
        """
        has_filter = any(
            v is not None
            for v in (
                workflow_id,
                parent_workflow_run_id,
                parent_step_run_id,
                statuses,
                additional_metadata,
            )
        )

        if not scheduled_ids and not has_filter:
            raise ValueError(
                "bulk_delete requires either scheduled_ids or at least one filter field."
            )

        return await asyncio.to_thread(
            self.bulk_delete,
            scheduled_ids=scheduled_ids,
            workflow_id=workflow_id,
            parent_workflow_run_id=parent_workflow_run_id,
            parent_step_run_id=parent_step_run_id,
            statuses=statuses,
            additional_metadata=additional_metadata,
        )

    def bulk_update(
        self,
        updates: (
            list[ScheduledWorkflowsBulkUpdateItem] | list[tuple[str, datetime.datetime]]
        ),
    ) -> ScheduledWorkflowsBulkUpdateResponse:
        """
        Bulk reschedule scheduled workflow runs.

        :param updates: Either:
          - a list of `(scheduled_id, trigger_at)` tuples, or
          - a list of `ScheduledWorkflowsBulkUpdateItem` objects
        :return: The bulk update response containing updated IDs and per-item errors.
        """
        update_items: list[ScheduledWorkflowsBulkUpdateItem] = []
        for u in updates:
            if isinstance(u, ScheduledWorkflowsBulkUpdateItem):
                update_items.append(u)
            else:
                scheduled_id, trigger_at = u
                update_items.append(
                    ScheduledWorkflowsBulkUpdateItem(
                        id=scheduled_id, triggerAt=trigger_at
                    )
                )

        with self.client() as client:
            return self._wa(client).workflow_scheduled_bulk_update(
                tenant=self.client_config.tenant_id,
                scheduled_workflows_bulk_update_request=ScheduledWorkflowsBulkUpdateRequest(
                    updates=update_items
                ),
            )

    async def aio_bulk_update(
        self,
        updates: (
            list[ScheduledWorkflowsBulkUpdateItem] | list[tuple[str, datetime.datetime]]
        ),
    ) -> ScheduledWorkflowsBulkUpdateResponse:
        """
        Bulk reschedule scheduled workflow runs.

        See `bulk_update` for parameter details.
        """
        return await asyncio.to_thread(self.bulk_update, updates)

    async def aio_list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        parent_workflow_run_id: str | None = None,
        statuses: list[ScheduledRunStatus] | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        order_by_field: ScheduledWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> ScheduledWorkflowsList:
        """
        Retrieves a list of scheduled workflows based on provided filters.

        :param offset: The offset to use in pagination.
        :param limit: The maximum number of scheduled workflows to return.
        :param workflow_id: The ID of the workflow to filter by.
        :param parent_workflow_run_id: The ID of the parent workflow run to filter by.
        :param statuses: A list of statuses to filter by.
        :param additional_metadata: Additional metadata to filter by.
        :param order_by_field: The field to order the results by.
        :param order_by_direction: The direction to order the results by.

        :return: A list of scheduled workflows matching the provided filters.
        """
        return await asyncio.to_thread(
            self.list,
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
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
        additional_metadata: JSONSerializableMapping | None = None,
        order_by_field: ScheduledWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> ScheduledWorkflowsList:
        """
        Retrieves a list of scheduled workflows based on provided filters.

        :param offset: The offset to use in pagination.
        :param limit: The maximum number of scheduled workflows to return.
        :param workflow_id: The ID of the workflow to filter by.
        :param parent_workflow_run_id: The ID of the parent workflow run to filter by.
        :param statuses: A list of statuses to filter by.
        :param additional_metadata: Additional metadata to filter by.
        :param order_by_field: The field to order the results by.
        :param order_by_direction: The direction to order the results by.

        :return: A list of scheduled workflows matching the provided filters.
        """
        with self.client() as client:
            workflow_scheduled_list = tenacity_retry(
                self._wa(client).workflow_scheduled_list, self.client_config.tenacity
            )
            return workflow_scheduled_list(
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

    def get(self, scheduled_id: str) -> ScheduledWorkflows:
        """
        Retrieves a specific scheduled workflow by scheduled run trigger ID.

        :param scheduled_id: The scheduled workflow trigger ID to retrieve.
        :return: The requested scheduled workflow instance.
        """

        with self.client() as client:
            workflow_scheduled_get = tenacity_retry(
                self._wa(client).workflow_scheduled_get, self.client_config.tenacity
            )
            return workflow_scheduled_get(
                tenant=self.client_config.tenant_id,
                scheduled_workflow_run=scheduled_id,
            )

    async def aio_get(self, scheduled_id: str) -> ScheduledWorkflows:
        """
        Retrieves a specific scheduled workflow by scheduled run trigger ID.

        :param scheduled_id: The scheduled workflow trigger ID to retrieve.
        :return: The requested scheduled workflow instance.
        """
        return await asyncio.to_thread(self.get, scheduled_id)
