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
    retry,
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

    @retry
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
            return self._wa(client).workflow_scheduled_list(
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

    @retry
    def get(self, scheduled_id: str) -> ScheduledWorkflows:
        """
        Retrieves a specific scheduled workflow by scheduled run trigger ID.

        :param scheduled_id: The scheduled workflow trigger ID to retrieve.
        :return: The requested scheduled workflow instance.
        """

        with self.client() as client:
            return self._wa(client).workflow_scheduled_get(
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
