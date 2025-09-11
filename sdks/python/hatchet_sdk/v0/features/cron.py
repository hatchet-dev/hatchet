from typing import Union

from pydantic import BaseModel, field_validator

from hatchet_sdk.v0.client import Client
from hatchet_sdk.v0.clients.rest.models.cron_workflows import CronWorkflows
from hatchet_sdk.v0.clients.rest.models.cron_workflows_list import CronWorkflowsList
from hatchet_sdk.v0.clients.rest.models.cron_workflows_order_by_field import (
    CronWorkflowsOrderByField,
)
from hatchet_sdk.v0.clients.rest.models.workflow_run_order_by_direction import (
    WorkflowRunOrderByDirection,
)


class CreateCronTriggerInput(BaseModel):
    """
    Schema for creating a workflow run triggered by a cron.

    Attributes:
        expression (str): The cron expression defining the schedule.
        input (dict): The input data for the cron workflow.
        additional_metadata (dict[str, str]): Additional metadata associated with the cron trigger.
    """

    expression: str = None
    input: dict = {}
    additional_metadata: dict[str, str] = {}

    @field_validator("expression")
    def validate_cron_expression(cls, v):
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


class CronClient:
    """
    Client for managing workflow cron triggers synchronously.

    Attributes:
        _client (Client): The underlying client used to interact with the REST API.
        aio (CronClientAsync): Asynchronous counterpart of CronClient.
    """

    _client: Client

    def __init__(self, _client: Client):
        """
        Initializes the CronClient with a given Client instance.

        Args:
            _client (Client): The client instance to be used for REST interactions.
        """
        self._client = _client
        self.aio = CronClientAsync(_client)

    def create(
        self,
        workflow_name: str,
        cron_name: str,
        expression: str,
        input: dict,
        additional_metadata: dict[str, str],
    ) -> CronWorkflows:
        """
        Creates a new workflow cron trigger.

        Args:
            workflow_name (str): The name of the workflow to trigger.
            cron_name (str): The name of the cron trigger.
            expression (str): The cron expression defining the schedule.
            input (dict): The input data for the cron workflow.
            additional_metadata (dict[str, str]): Additional metadata associated with the cron trigger.

        Returns:
            CronWorkflows: The created cron workflow instance.
        """
        validated_input = CreateCronTriggerInput(
            expression=expression, input=input, additional_metadata=additional_metadata
        )

        return self._client.rest.cron_create(
            workflow_name,
            cron_name,
            validated_input.expression,
            validated_input.input,
            validated_input.additional_metadata,
        )

    def delete(self, cron_trigger: Union[str, CronWorkflows]) -> None:
        """
        Deletes a workflow cron trigger.

        Args:
            cron_trigger (Union[str, CronWorkflows]): The cron trigger ID or CronWorkflows instance to delete.
        """
        id_ = cron_trigger
        if isinstance(cron_trigger, CronWorkflows):
            id_ = cron_trigger.metadata.id
        self._client.rest.cron_delete(id_)

    def list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        additional_metadata: list[str] | None = None,
        order_by_field: CronWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> CronWorkflowsList:
        """
        Retrieves a list of all workflow cron triggers matching the criteria.

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
        return self._client.rest.cron_list(
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
        )

    def get(self, cron_trigger: Union[str, CronWorkflows]) -> CronWorkflows:
        """
        Retrieves a specific workflow cron trigger by ID.

        Args:
            cron_trigger (Union[str, CronWorkflows]): The cron trigger ID or CronWorkflows instance to retrieve.

        Returns:
            CronWorkflows: The requested cron workflow instance.
        """
        id_ = cron_trigger
        if isinstance(cron_trigger, CronWorkflows):
            id_ = cron_trigger.metadata.id
        return self._client.rest.cron_get(id_)


class CronClientAsync:
    """
    Asynchronous client for managing workflow cron triggers.

    Attributes:
        _client (Client): The underlying client used to interact with the REST API asynchronously.
    """

    _client: Client

    def __init__(self, _client: Client):
        """
        Initializes the CronClientAsync with a given Client instance.

        Args:
            _client (Client): The client instance to be used for asynchronous REST interactions.
        """
        self._client = _client

    async def create(
        self,
        workflow_name: str,
        cron_name: str,
        expression: str,
        input: dict,
        additional_metadata: dict[str, str],
    ) -> CronWorkflows:
        """
        Asynchronously creates a new workflow cron trigger.

        Args:
            workflow_name (str): The name of the workflow to trigger.
            cron_name (str): The name of the cron trigger.
            expression (str): The cron expression defining the schedule.
            input (dict): The input data for the cron workflow.
            additional_metadata (dict[str, str]): Additional metadata associated with the cron trigger.

        Returns:
            CronWorkflows: The created cron workflow instance.
        """
        validated_input = CreateCronTriggerInput(
            expression=expression, input=input, additional_metadata=additional_metadata
        )

        return await self._client.rest.aio.cron_create(
            workflow_name=workflow_name,
            cron_name=cron_name,
            expression=validated_input.expression,
            input=validated_input.input,
            additional_metadata=validated_input.additional_metadata,
        )

    async def delete(self, cron_trigger: Union[str, CronWorkflows]) -> None:
        """
        Asynchronously deletes a workflow cron trigger.

        Args:
            cron_trigger (Union[str, CronWorkflows]): The cron trigger ID or CronWorkflows instance to delete.
        """
        id_ = cron_trigger
        if isinstance(cron_trigger, CronWorkflows):
            id_ = cron_trigger.metadata.id
        await self._client.rest.aio.cron_delete(id_)

    async def list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        additional_metadata: list[str] | None = None,
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
        return await self._client.rest.aio.cron_list(
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
        )

    async def get(self, cron_trigger: Union[str, CronWorkflows]) -> CronWorkflows:
        """
        Asynchronously retrieves a specific workflow cron trigger by ID.

        Args:
            cron_trigger (Union[str, CronWorkflows]): The cron trigger ID or CronWorkflows instance to retrieve.

        Returns:
            CronWorkflows: The requested cron workflow instance.
        """
        id_ = cron_trigger
        if isinstance(cron_trigger, CronWorkflows):
            id_ = cron_trigger.metadata.id
        return await self._client.rest.aio.cron_get(id_)
