import asyncio

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
    retry,
)
from hatchet_sdk.utils.typing import JSONSerializableMapping


class CreateCronTriggerConfig(BaseModel):
    """
    Schema for creating a workflow run triggered by a cron.
    """

    expression: str
    input: JSONSerializableMapping = Field(default_factory=dict)
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)

    @field_validator("expression")
    @classmethod
    def validate_cron_expression(cls, v: str) -> str:
        """
        Validates the cron expression to ensure it adheres to the expected format.

        :param v: The cron expression to validate.

        :raises ValueError: If the expression is invalid

        :return: The validated cron expression.
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
    """
    The cron client is a client for managing cron workflows within Hatchet.
    """

    def _wra(self, client: ApiClient) -> WorkflowRunApi:
        return WorkflowRunApi(client)

    def _wa(self, client: ApiClient) -> WorkflowApi:
        return WorkflowApi(client)

    def create(
        self,
        workflow_name: str,
        cron_name: str,
        expression: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping,
        priority: int | None = None,
    ) -> CronWorkflows:
        """
        Create a new workflow cron trigger.

        :param workflow_name: The name of the workflow to trigger.
        :param cron_name: The name of the cron trigger.
        :param expression: The cron expression defining the schedule.
        :param input: The input data for the cron workflow.
        :param additional_metadata: Additional metadata associated with the cron trigger.
        :param priority: The priority of the cron workflow trigger.

        :return: The created cron workflow instance.
        """
        validated_input = CreateCronTriggerConfig(
            expression=expression, input=input, additional_metadata=additional_metadata
        )

        with self.client() as client:
            return self._wra(client).cron_workflow_trigger_create(
                tenant=self.client_config.tenant_id,
                workflow=self.client_config.apply_namespace(workflow_name),
                create_cron_workflow_trigger_request=CreateCronWorkflowTriggerRequest(
                    cronName=cron_name,
                    cronExpression=validated_input.expression,
                    input=validated_input.input,
                    additionalMetadata=validated_input.additional_metadata,
                    priority=priority,
                ),
            )

    async def aio_create(
        self,
        workflow_name: str,
        cron_name: str,
        expression: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping,
        priority: int | None = None,
    ) -> CronWorkflows:
        """
        Create a new workflow cron trigger.

        :param workflow_name: The name of the workflow to trigger.
        :param cron_name: The name of the cron trigger.
        :param expression: The cron expression defining the schedule.
        :param input: The input data for the cron workflow.
        :param additional_metadata: Additional metadata associated with the cron trigger.
        :param priority: The priority of the cron workflow trigger.

        :return: The created cron workflow instance.
        """
        return await asyncio.to_thread(
            self.create,
            workflow_name,
            cron_name,
            expression,
            input,
            additional_metadata,
            priority,
        )

    def delete(self, cron_id: str) -> None:
        """
        Delete a workflow cron trigger.

        :param cron_id: The ID of the cron trigger to delete.
        :return: None
        """
        with self.client() as client:
            self._wa(client).workflow_cron_delete(
                tenant=self.client_config.tenant_id, cron_workflow=str(cron_id)
            )

    async def aio_delete(self, cron_id: str) -> None:
        """
        Delete a workflow cron trigger.

        :param cron_id: The ID of the cron trigger to delete.
        :return: None
        """
        return await asyncio.to_thread(self.delete, cron_id)

    async def aio_list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        order_by_field: CronWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
        workflow_name: str | None = None,
        cron_name: str | None = None,
    ) -> CronWorkflowsList:
        """
        Retrieve a list of all workflow cron triggers matching the criteria.

        :param offset: The offset to start the list from.
        :param limit: The maximum number of items to return.
        :param workflow_id: The ID of the workflow to filter by.
        :param additional_metadata: Filter by additional metadata keys.
        :param order_by_field: The field to order the list by.
        :param order_by_direction: The direction to order the list by.
        :param workflow_name: The name of the workflow to filter by.
        :param cron_name: The name of the cron trigger to filter by.

        :return: A list of cron workflows.
        """
        return await asyncio.to_thread(
            self.list,
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
            workflow_name=workflow_name,
            cron_name=cron_name,
        )

    @retry
    def list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        order_by_field: CronWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
        workflow_name: str | None = None,
        cron_name: str | None = None,
    ) -> CronWorkflowsList:
        """
        Retrieve a list of all workflow cron triggers matching the criteria.

        :param offset: The offset to start the list from.
        :param limit: The maximum number of items to return.
        :param workflow_id: The ID of the workflow to filter by.
        :param additional_metadata: Filter by additional metadata keys.
        :param order_by_field: The field to order the list by.
        :param order_by_direction: The direction to order the list by.
        :param workflow_name: The name of the workflow to filter by.
        :param cron_name: The name of the cron trigger to filter by.

        :return: A list of cron workflows.
        """
        with self.client() as client:
            return self._wa(client).cron_workflow_list(
                tenant=self.client_config.tenant_id,
                offset=offset,
                limit=limit,
                workflow_id=workflow_id,
                additional_metadata=maybe_additional_metadata_to_kv(
                    additional_metadata
                ),
                order_by_field=order_by_field,
                order_by_direction=order_by_direction,
                workflow_name=workflow_name,
                cron_name=cron_name,
            )

    @retry
    def get(self, cron_id: str) -> CronWorkflows:
        """
        Retrieve a specific workflow cron trigger by ID.

        :param cron_id: The cron trigger ID or CronWorkflows instance to retrieve.
        :return: The requested cron workflow instance.
        """
        with self.client() as client:
            return self._wa(client).workflow_cron_get(
                tenant=self.client_config.tenant_id, cron_workflow=str(cron_id)
            )

    async def aio_get(self, cron_id: str) -> CronWorkflows:
        """
        Retrieve a specific workflow cron trigger by ID.

        :param cron_id: The cron trigger ID or CronWorkflows instance to retrieve.
        :return: The requested cron workflow instance.
        """
        return await asyncio.to_thread(self.get, cron_id)
