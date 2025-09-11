import datetime
from typing import Any, Coroutine, Dict, List, Optional, Union

from pydantic import BaseModel

from hatchet_sdk.v0.client import Client
from hatchet_sdk.v0.clients.rest.models.cron_workflows import CronWorkflows
from hatchet_sdk.v0.clients.rest.models.cron_workflows_order_by_field import (
    CronWorkflowsOrderByField,
)
from hatchet_sdk.v0.clients.rest.models.scheduled_workflows import ScheduledWorkflows
from hatchet_sdk.v0.clients.rest.models.scheduled_workflows_list import (
    ScheduledWorkflowsList,
)
from hatchet_sdk.v0.clients.rest.models.workflow_run_order_by_direction import (
    WorkflowRunOrderByDirection,
)


class CreateScheduledTriggerInput(BaseModel):
    """
    Schema for creating a scheduled workflow run.

    Attributes:
        input (Dict[str, Any]): The input data for the scheduled workflow.
        additional_metadata (Dict[str, str]): Additional metadata associated with the future run (e.g. ["key1:value1", "key2:value2"]).
        trigger_at (Optional[datetime.datetime]): The datetime when the run should be triggered.
    """

    input: Dict[str, Any] = {}
    additional_metadata: Dict[str, str] = {}
    trigger_at: Optional[datetime.datetime] = None


class ScheduledClient:
    """
    Client for managing scheduled workflows synchronously.

    Attributes:
        _client (Client): The underlying client used to interact with the REST API.
        aio (ScheduledClientAsync): Asynchronous counterpart of ScheduledClient.
    """

    _client: Client

    def __init__(self, _client: Client) -> None:
        """
        Initializes the ScheduledClient with a given Client instance.

        Args:
            _client (Client): The client instance to be used for REST interactions.
        """
        self._client = _client
        self.aio: "ScheduledClientAsync" = ScheduledClientAsync(_client)

    def create(
        self,
        workflow_name: str,
        trigger_at: datetime.datetime,
        input: Dict[str, Any],
        additional_metadata: Dict[str, str],
    ) -> ScheduledWorkflows:
        """
        Creates a new scheduled workflow run asynchronously.

        Args:
            workflow_name (str): The name of the scheduled workflow.
            trigger_at (datetime.datetime): The datetime when the run should be triggered.
            input (Dict[str, Any]): The input data for the scheduled workflow.
            additional_metadata (Dict[str, str]): Additional metadata associated with the future run as a key-value pair.

        Returns:
            ScheduledWorkflows: The created scheduled workflow instance.
        """

        validated_input = CreateScheduledTriggerInput(
            trigger_at=trigger_at, input=input, additional_metadata=additional_metadata
        )

        return self._client.rest.schedule_create(
            workflow_name,
            validated_input.trigger_at,
            validated_input.input,
            validated_input.additional_metadata,
        )

    def delete(self, scheduled: Union[str, ScheduledWorkflows]) -> None:
        """
        Deletes a scheduled workflow run.

        Args:
            scheduled (Union[str, ScheduledWorkflows]): The scheduled workflow trigger ID or ScheduledWorkflows instance to delete.
        """
        id_ = scheduled
        if isinstance(scheduled, ScheduledWorkflows):
            id_ = scheduled.metadata.id
        self._client.rest.schedule_delete(id_)

    def list(
        self,
        offset: Optional[int] = None,
        limit: Optional[int] = None,
        workflow_id: Optional[str] = None,
        additional_metadata: Optional[List[str]] = None,
        order_by_field: Optional[CronWorkflowsOrderByField] = None,
        order_by_direction: Optional[WorkflowRunOrderByDirection] = None,
    ) -> ScheduledWorkflowsList:
        """
        Retrieves a list of scheduled workflows based on provided filters.

        Args:
            offset (Optional[int]): The starting point for the list.
            limit (Optional[int]): The maximum number of items to return.
            workflow_id (Optional[str]): Filter by specific workflow ID.
            additional_metadata (Optional[List[str]]): Filter by additional metadata keys (e.g. ["key1:value1", "key2:value2"]).
            order_by_field (Optional[CronWorkflowsOrderByField]): Field to order the results by.
            order_by_direction (Optional[WorkflowRunOrderByDirection]): Direction to order the results.

        Returns:
            List[ScheduledWorkflows]: A list of scheduled workflows matching the criteria.
        """
        return self._client.rest.schedule_list(
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
        )

    def get(self, scheduled: Union[str, ScheduledWorkflows]) -> ScheduledWorkflows:
        """
        Retrieves a specific scheduled workflow by scheduled run trigger ID.

        Args:
            scheduled (Union[str, ScheduledWorkflows]): The scheduled workflow trigger ID or ScheduledWorkflows instance to retrieve.

        Returns:
            ScheduledWorkflows: The requested scheduled workflow instance.
        """
        id_ = scheduled
        if isinstance(scheduled, ScheduledWorkflows):
            id_ = scheduled.metadata.id
        return self._client.rest.schedule_get(id_)


class ScheduledClientAsync:
    """
    Asynchronous client for managing scheduled workflows.

    Attributes:
        _client (Client): The underlying client used to interact with the REST API asynchronously.
    """

    _client: Client

    def __init__(self, _client: Client) -> None:
        """
        Initializes the ScheduledClientAsync with a given Client instance.

        Args:
            _client (Client): The client instance to be used for asynchronous REST interactions.
        """
        self._client = _client

    async def create(
        self,
        workflow_name: str,
        trigger_at: datetime.datetime,
        input: Dict[str, Any],
        additional_metadata: Dict[str, str],
    ) -> ScheduledWorkflows:
        """
        Creates a new scheduled workflow run asynchronously.

        Args:
            workflow_name (str): The name of the scheduled workflow.
            trigger_at (datetime.datetime): The datetime when the run should be triggered.
            input (Dict[str, Any]): The input data for the scheduled workflow.
            additional_metadata (Dict[str, str]): Additional metadata associated with the future run.

        Returns:
            ScheduledWorkflows: The created scheduled workflow instance.
        """
        return await self._client.rest.aio.schedule_create(
            workflow_name, trigger_at, input, additional_metadata
        )

    async def delete(self, scheduled: Union[str, ScheduledWorkflows]) -> None:
        """
        Deletes a scheduled workflow asynchronously.

        Args:
            scheduled (Union[str, ScheduledWorkflows]): The scheduled workflow trigger ID or ScheduledWorkflows instance to delete.
        """
        id_ = scheduled
        if isinstance(scheduled, ScheduledWorkflows):
            id_ = scheduled.metadata.id
        await self._client.rest.aio.schedule_delete(id_)

    async def list(
        self,
        offset: Optional[int] = None,
        limit: Optional[int] = None,
        workflow_id: Optional[str] = None,
        additional_metadata: Optional[List[str]] = None,
        order_by_field: Optional[CronWorkflowsOrderByField] = None,
        order_by_direction: Optional[WorkflowRunOrderByDirection] = None,
    ) -> ScheduledWorkflowsList:
        """
        Retrieves a list of scheduled workflows based on provided filters asynchronously.

        Args:
            offset (Optional[int]): The starting point for the list.
            limit (Optional[int]): The maximum number of items to return.
            workflow_id (Optional[str]): Filter by specific workflow ID.
            additional_metadata (Optional[List[str]]): Filter by additional metadata keys (e.g. ["key1:value1", "key2:value2"]).
            order_by_field (Optional[CronWorkflowsOrderByField]): Field to order the results by.
            order_by_direction (Optional[WorkflowRunOrderByDirection]): Direction to order the results.

        Returns:
            ScheduledWorkflowsList: A list of scheduled workflows matching the criteria.
        """
        return await self._client.rest.aio.schedule_list(
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
        )

    async def get(
        self, scheduled: Union[str, ScheduledWorkflows]
    ) -> ScheduledWorkflows:
        """
        Retrieves a specific scheduled workflow by scheduled run trigger ID asynchronously.

        Args:
            scheduled (Union[str, ScheduledWorkflows]): The scheduled workflow trigger ID or ScheduledWorkflows instance to retrieve.

        Returns:
            ScheduledWorkflows: The requested scheduled workflow instance.
        """
        id_ = scheduled
        if isinstance(scheduled, ScheduledWorkflows):
            id_ = scheduled.metadata.id
        return await self._client.rest.aio.schedule_get(id_)
