from pydantic import BaseModel, Field, field_validator

from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi
from hatchet_sdk.clients.rest.api.workflow_run_api import WorkflowRunApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.create_cron_workflow_trigger_request import (
    CreateCronWorkflowTriggerRequest,
)
from hatchet_sdk.clients.rest.models.cron_workflows import CronWorkflows
from hatchet_sdk.clients.rest.models.cron_workflows_list import CronWorkflowsList
from hatchet_sdk.clients.rest.models.cron_workflows_order_by_field import (
    CronWorkflowsOrderByField,
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


class CreateCronTriggerConfig(BaseModel):
    """
    Schema for creating a workflow run triggered by a cron.

    Attributes:
        expression (str): The cron expression defining the schedule.
        input (dict): The input data for the cron workflow.
        additional_metadata (dict[str, str]): Additional metadata associated with the cron trigger (e.g. {"key1": "value1", "key2": "value2"}).
    """

    expression: str
    input: JSONSerializableMapping = Field(default_factory=dict)
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)

    @field_validator("expression")
    @classmethod
    def validate_cron_expression(cls, v: str) -> str:
        """
        Validates the cron expression to ensure it adheres to the expected format.

        Args:
            v (str): The cron expression to validate.

        Raises:
            ValueError: If the expression is invalid.

        Returns:
            str: The validated cron expression.
        """
        if not v:
            raise ValueError("Cron expression is required")

        parts = v.split()
        if len(parts) != 5:
            raise ValueError(
                "Cron expression must have 5 parts: minute hour day month weekday"
            )

        for part in parts:
            if not (
                part == "*"
                or part.replace("*/", "").replace("-", "").replace(",", "").isdigit()
            ):
                raise ValueError(f"Invalid cron expression part: {part}")

        return v


class CronClient(BaseRestClient):
    def _wra(self, client: ApiClient) -> WorkflowRunApi:
        return WorkflowRunApi(client)

    def _wa(self, client: ApiClient) -> WorkflowApi:
        return WorkflowApi(client)

    async def aio_create(
        self,
        workflow_name: str,
        cron_name: str,
        expression: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping,
    ) -> CronWorkflows:
        """
        Asynchronously creates a new workflow cron trigger.

        Args:
            workflow_name (str): The name of the workflow to trigger.
            cron_name (str): The name of the cron trigger.
            expression (str): The cron expression defining the schedule.
            input (dict): The input data for the cron workflow.
            additional_metadata (dict[str, str]): Additional metadata associated with the cron trigger (e.g. {"key1": "value1", "key2": "value2"}).

        Returns:
            CronWorkflows: The created cron workflow instance.
        """
        validated_input = CreateCronTriggerConfig(
            expression=expression, input=input, additional_metadata=additional_metadata
        )

        async with self.client() as client:
            return await self._wra(client).cron_workflow_trigger_create(
                tenant=self.client_config.tenant_id,
                workflow=workflow_name,
                create_cron_workflow_trigger_request=CreateCronWorkflowTriggerRequest(
                    cronName=cron_name,
                    cronExpression=validated_input.expression,
                    input=dict(validated_input.input),
                    additionalMetadata=dict(validated_input.additional_metadata),
                ),
            )

    def create(
        self,
        workflow_name: str,
        cron_name: str,
        expression: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping,
    ) -> CronWorkflows:
        return run_async_from_sync(
            self.aio_create,
            workflow_name,
            cron_name,
            expression,
            input,
            additional_metadata,
        )

    async def aio_delete(self, cron_id: str) -> None:
        """
        Asynchronously deletes a workflow cron trigger.

        Args:
            cron_id (str): The cron trigger ID or CronWorkflows instance to delete.
        """
        async with self.client() as client:
            await self._wa(client).workflow_cron_delete(
                tenant=self.client_config.tenant_id, cron_workflow=str(cron_id)
            )

    def delete(self, cron_id: str) -> None:
        return run_async_from_sync(self.aio_delete, cron_id)

    async def aio_list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        order_by_field: CronWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> CronWorkflowsList:
        """
        Asynchronously retrieves a list of all workflow cron triggers matching the criteria.

        Args:
            offset (int | None): The offset to start the list from.
            limit (int | None): The maximum number of items to return.
            workflow_id (str | None): The ID of the workflow to filter by.
            additional_metadata (list[str] | None): Filter by additional metadata keys (e.g. ["key1:value1", "key2:value2"]).
            order_by_field (CronWorkflowsOrderByField | None): The field to order the list by.
            order_by_direction (WorkflowRunOrderByDirection | None): The direction to order the list by.

        Returns:
            CronWorkflowsList: A list of cron workflows.
        """
        async with self.client() as client:
            return await self._wa(client).cron_workflow_list(
                tenant=self.client_config.tenant_id,
                offset=offset,
                limit=limit,
                workflow_id=workflow_id,
                additional_metadata=maybe_additional_metadata_to_kv(
                    additional_metadata
                ),
                order_by_field=order_by_field,
                order_by_direction=order_by_direction,
            )

    def list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        order_by_field: CronWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> CronWorkflowsList:
        """
        Synchronously retrieves a list of all workflow cron triggers matching the criteria.

        Args:
            offset (int | None): The offset to start the list from.
            limit (int | None): The maximum number of items to return.
            workflow_id (str | None): The ID of the workflow to filter by.
            additional_metadata (list[str] | None): Filter by additional metadata keys (e.g. ["key1:value1", "key2:value2"]).
            order_by_field (CronWorkflowsOrderByField | None): The field to order the list by.
            order_by_direction (WorkflowRunOrderByDirection | None): The direction to order the list by.

        Returns:
            CronWorkflowsList: A list of cron workflows.
        """
        return run_async_from_sync(
            self.aio_list,
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
        )

    async def aio_get(self, cron_id: str) -> CronWorkflows:
        """
        Asynchronously retrieves a specific workflow cron trigger by ID.

        Args:
            cron_id (str): The cron trigger ID or CronWorkflows instance to retrieve.

        Returns:
            CronWorkflows: The requested cron workflow instance.
        """
        async with self.client() as client:
            return await self._wa(client).workflow_cron_get(
                tenant=self.client_config.tenant_id, cron_workflow=str(cron_id)
            )

    def get(self, cron_id: str) -> CronWorkflows:
        """
        Synchronously retrieves a specific workflow cron trigger by ID.

        Args:
            cron_id (str): The cron trigger ID or CronWorkflows instance to retrieve.

        Returns:
            CronWorkflows: The requested cron workflow instance.
        """
        return run_async_from_sync(self.aio_get, cron_id)
