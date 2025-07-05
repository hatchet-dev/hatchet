import asyncio
from collections.abc import AsyncIterator
from datetime import datetime, timedelta, timezone
from typing import TYPE_CHECKING, Literal, overload
from warnings import warn

from pydantic import BaseModel, model_validator

from hatchet_sdk.clients.listeners.run_event_listener import (
    RunEventListenerClient,
    StepRunEventType,
)
from hatchet_sdk.clients.listeners.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.clients.rest.api.task_api import TaskApi
from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_cancel_task_request import V1CancelTaskRequest
from hatchet_sdk.clients.rest.models.v1_replay_task_request import V1ReplayTaskRequest
from hatchet_sdk.clients.rest.models.v1_task_filter import V1TaskFilter
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
from hatchet_sdk.clients.rest.models.v1_task_summary_list import V1TaskSummaryList
from hatchet_sdk.clients.rest.models.v1_trigger_workflow_run_request import (
    V1TriggerWorkflowRunRequest,
)
from hatchet_sdk.clients.rest.models.v1_workflow_run_details import V1WorkflowRunDetails
from hatchet_sdk.clients.v1.api_client import (
    BaseRestClient,
    maybe_additional_metadata_to_kv,
    retry,
)
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.utils.aio import gather_max_concurrency
from hatchet_sdk.utils.datetimes import partition_date_range
from hatchet_sdk.utils.typing import JSONSerializableMapping

if TYPE_CHECKING:
    from hatchet_sdk.workflow_run import WorkflowRunRef


class RunFilter(BaseModel):
    since: datetime
    until: datetime | None = None
    statuses: list[V1TaskStatus] | None = None
    workflow_ids: list[str] | None = None
    additional_metadata: dict[str, str] | None = None


class BulkCancelReplayOpts(BaseModel):
    ids: list[str] | None = None
    filters: RunFilter | None = None

    @model_validator(mode="after")
    def validate_model(self) -> "BulkCancelReplayOpts":
        if not self.ids and not self.filters:
            raise ValueError("ids or filters must be set")

        if self.ids and self.filters:
            raise ValueError("ids and filters cannot both be set")

        return self

    @property
    def v1_task_filter(self) -> V1TaskFilter | None:
        if not self.filters:
            return None

        return V1TaskFilter(
            since=self.filters.since,
            until=self.filters.until,
            statuses=self.filters.statuses,
            workflowIds=self.filters.workflow_ids,
            additionalMetadata=maybe_additional_metadata_to_kv(
                self.filters.additional_metadata
            ),
        )

    @overload
    def to_request(self, request_type: Literal["replay"]) -> V1ReplayTaskRequest: ...

    @overload
    def to_request(self, request_type: Literal["cancel"]) -> V1CancelTaskRequest: ...

    def to_request(
        self, request_type: Literal["replay", "cancel"]
    ) -> V1ReplayTaskRequest | V1CancelTaskRequest:
        if request_type == "replay":
            return V1ReplayTaskRequest(
                externalIds=self.ids,
                filter=self.v1_task_filter,
            )

        if request_type == "cancel":
            return V1CancelTaskRequest(
                externalIds=self.ids,
                filter=self.v1_task_filter,
            )


class RunsClient(BaseRestClient):
    """
    The runs client is a client for interacting with task and workflow runs within Hatchet.
    """

    def __init__(
        self,
        config: ClientConfig,
        workflow_run_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
    ) -> None:
        super().__init__(config)

        self.workflow_run_listener = workflow_run_listener
        self.workflow_run_event_listener = workflow_run_event_listener

    def _wra(self, client: ApiClient) -> WorkflowRunsApi:
        return WorkflowRunsApi(client)

    def _ta(self, client: ApiClient) -> TaskApi:
        return TaskApi(client)

    @retry
    def get(self, workflow_run_id: str) -> V1WorkflowRunDetails:
        """
        Get workflow run details for a given workflow run ID.

        :param workflow_run_id: The ID of the workflow run to retrieve details for.
        :return: Workflow run details for the specified workflow run ID.
        """
        with self.client() as client:
            return self._wra(client).v1_workflow_run_get(str(workflow_run_id))

    async def aio_get(self, workflow_run_id: str) -> V1WorkflowRunDetails:
        """
        Get workflow run details for a given workflow run ID.

        :param workflow_run_id: The ID of the workflow run to retrieve details for.
        :return: Workflow run details for the specified workflow run ID.
        """
        return await asyncio.to_thread(self.get, workflow_run_id)

    @retry
    def get_status(self, workflow_run_id: str) -> V1TaskStatus:
        """
        Get workflow run status for a given workflow run ID.

        :param workflow_run_id: The ID of the workflow run to retrieve details for.
        :return: The task status
        """
        with self.client() as client:
            return self._wra(client).v1_workflow_run_get_status(str(workflow_run_id))

    async def aio_get_status(self, workflow_run_id: str) -> V1TaskStatus:
        """
        Get workflow run status for a given workflow run ID.

        :param workflow_run_id: The ID of the workflow run to retrieve details for.
        :return: The task status
        """
        return await asyncio.to_thread(self.get_status, workflow_run_id)

    @retry
    def list_with_pagination(
        self,
        since: datetime | None = None,
        only_tasks: bool = False,
        offset: int | None = None,
        limit: int | None = None,
        statuses: list[V1TaskStatus] | None = None,
        until: datetime | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
        worker_id: str | None = None,
        parent_task_external_id: str | None = None,
        triggering_event_external_id: str | None = None,
    ) -> list[V1TaskSummary]:
        """
        List task runs according to a set of filters, paginating through days

        :param since: The start time for filtering task runs.
        :param only_tasks: Whether to only list task runs.
        :param offset: The offset for pagination.
        :param limit: The maximum number of task runs to return.
        :param statuses: The statuses to filter task runs by.
        :param until: The end time for filtering task runs.
        :param additional_metadata: Additional metadata to filter task runs by.
        :param workflow_ids: The workflow IDs to filter task runs by.
        :param worker_id: The worker ID to filter task runs by.
        :param parent_task_external_id: The parent task external ID to filter task runs by.
        :param triggering_event_external_id: The event id that triggered the task run.

        :return: A list of task runs matching the specified filters.
        """

        date_ranges = partition_date_range(
            since=since or datetime.now(tz=timezone.utc) - timedelta(days=1),
            until=until or datetime.now(tz=timezone.utc),
        )

        with self.client() as client:
            responses = [
                self._wra(client).v1_workflow_run_list(
                    tenant=self.client_config.tenant_id,
                    since=s,
                    until=u,
                    only_tasks=only_tasks,
                    offset=offset,
                    limit=limit,
                    statuses=statuses,
                    additional_metadata=maybe_additional_metadata_to_kv(
                        additional_metadata
                    ),
                    workflow_ids=workflow_ids,
                    worker_id=worker_id,
                    parent_task_external_id=parent_task_external_id,
                    triggering_event_external_id=triggering_event_external_id,
                )
                for s, u in date_ranges
            ]

            ## Hack for uniqueness
            run_id_to_run = {
                run.metadata.id: run for record in responses for run in record.rows
            }

            return sorted(
                run_id_to_run.values(),
                key=lambda x: x.created_at,
                reverse=True,
            )

    @retry
    async def aio_list_with_pagination(
        self,
        since: datetime | None = None,
        only_tasks: bool = False,
        offset: int | None = None,
        limit: int | None = None,
        statuses: list[V1TaskStatus] | None = None,
        until: datetime | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
        worker_id: str | None = None,
        parent_task_external_id: str | None = None,
        triggering_event_external_id: str | None = None,
    ) -> list[V1TaskSummary]:
        """
        List task runs according to a set of filters, paginating through days

        :param since: The start time for filtering task runs.
        :param only_tasks: Whether to only list task runs.
        :param offset: The offset for pagination.
        :param limit: The maximum number of task runs to return.
        :param statuses: The statuses to filter task runs by.
        :param until: The end time for filtering task runs.
        :param additional_metadata: Additional metadata to filter task runs by.
        :param workflow_ids: The workflow IDs to filter task runs by.
        :param worker_id: The worker ID to filter task runs by.
        :param parent_task_external_id: The parent task external ID to filter task runs by.
        :param triggering_event_external_id: The event id that triggered the task run.

        :return: A list of task runs matching the specified filters.
        """

        date_ranges = partition_date_range(
            since=since or datetime.now(tz=timezone.utc) - timedelta(days=1),
            until=until or datetime.now(tz=timezone.utc),
        )

        with self.client() as client:
            coros = [
                asyncio.to_thread(
                    self._wra(client).v1_workflow_run_list,
                    tenant=self.client_config.tenant_id,
                    since=s,
                    until=u,
                    only_tasks=only_tasks,
                    offset=offset,
                    limit=limit,
                    statuses=statuses,
                    additional_metadata=maybe_additional_metadata_to_kv(
                        additional_metadata
                    ),
                    workflow_ids=workflow_ids,
                    worker_id=worker_id,
                    parent_task_external_id=parent_task_external_id,
                    triggering_event_external_id=triggering_event_external_id,
                )
                for s, u in date_ranges
            ]

            responses = await gather_max_concurrency(
                *coros,
                max_concurrency=3,
            )

            ## Hack for uniqueness
            run_id_to_run = {
                run.metadata.id: run for record in responses for run in record.rows
            }

            return sorted(
                run_id_to_run.values(),
                key=lambda x: x.created_at,
                reverse=True,
            )

    async def aio_list(
        self,
        since: datetime | None = None,
        only_tasks: bool = False,
        offset: int | None = None,
        limit: int | None = None,
        statuses: list[V1TaskStatus] | None = None,
        until: datetime | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
        worker_id: str | None = None,
        parent_task_external_id: str | None = None,
        triggering_event_external_id: str | None = None,
    ) -> V1TaskSummaryList:
        """
        List task runs according to a set of filters.

        :param since: The start time for filtering task runs.
        :param only_tasks: Whether to only list task runs.
        :param offset: The offset for pagination.
        :param limit: The maximum number of task runs to return.
        :param statuses: The statuses to filter task runs by.
        :param until: The end time for filtering task runs.
        :param additional_metadata: Additional metadata to filter task runs by.
        :param workflow_ids: The workflow IDs to filter task runs by.
        :param worker_id: The worker ID to filter task runs by.
        :param parent_task_external_id: The parent task external ID to filter task runs by.
        :param triggering_event_external_id: The event id that triggered the task run.

        :return: A list of task runs matching the specified filters.
        """
        return await asyncio.to_thread(
            self.list,
            since=since,
            only_tasks=only_tasks,
            offset=offset,
            limit=limit,
            statuses=statuses,
            until=until,
            additional_metadata=additional_metadata,
            workflow_ids=workflow_ids,
            worker_id=worker_id,
            parent_task_external_id=parent_task_external_id,
            triggering_event_external_id=triggering_event_external_id,
        )

    @retry
    def list(
        self,
        since: datetime | None = None,
        only_tasks: bool = False,
        offset: int | None = None,
        limit: int | None = None,
        statuses: list[V1TaskStatus] | None = None,
        until: datetime | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
        worker_id: str | None = None,
        parent_task_external_id: str | None = None,
        triggering_event_external_id: str | None = None,
    ) -> V1TaskSummaryList:
        """
        List task runs according to a set of filters.

        :param since: The start time for filtering task runs.
        :param only_tasks: Whether to only list task runs.
        :param offset: The offset for pagination.
        :param limit: The maximum number of task runs to return.
        :param statuses: The statuses to filter task runs by.
        :param until: The end time for filtering task runs.
        :param additional_metadata: Additional metadata to filter task runs by.
        :param workflow_ids: The workflow IDs to filter task runs by.
        :param worker_id: The worker ID to filter task runs by.
        :param parent_task_external_id: The parent task external ID to filter task runs by.
        :param triggering_event_external_id: The event id that triggered the task run.

        :return: A list of task runs matching the specified filters.
        """

        since = since or datetime.now(tz=timezone.utc) - timedelta(days=1)
        until = until or datetime.now(tz=timezone.utc)

        if (until - since).days >= 7:
            warn(
                "Listing runs with a date range longer than 7 days may result in performance issues. "
                "Consider using `list_with_pagination` or `aio_list_with_pagination` instead.",
                RuntimeWarning,
                stacklevel=2,
            )

        with self.client() as client:
            return self._wra(client).v1_workflow_run_list(
                tenant=self.client_config.tenant_id,
                since=since,
                only_tasks=only_tasks,
                offset=offset,
                limit=limit,
                statuses=statuses,
                until=until,
                additional_metadata=maybe_additional_metadata_to_kv(
                    additional_metadata
                ),
                workflow_ids=workflow_ids,
                worker_id=worker_id,
                parent_task_external_id=parent_task_external_id,
                triggering_event_external_id=triggering_event_external_id,
            )

    def create(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
    ) -> V1WorkflowRunDetails:
        """
        Trigger a new workflow run.

        IMPORTANT: It's preferable to use `Workflow.run` (and similar) to trigger workflows if possible. This method is intended to be an escape hatch. For more details, see [the documentation](https://docs.hatchet.run/sdks/python/runnables#workflow).

        :param workflow_name: The name of the workflow to trigger.
        :param input: The input data for the workflow run.
        :param additional_metadata: Additional metadata associated with the workflow run.
        :param priority: The priority of the workflow run.

        :return: The details of the triggered workflow run.
        """
        with self.client() as client:
            return self._wra(client).v1_workflow_run_create(
                tenant=self.client_config.tenant_id,
                v1_trigger_workflow_run_request=V1TriggerWorkflowRunRequest(
                    workflowName=self.client_config.apply_namespace(workflow_name),
                    input=input,
                    additionalMetadata=additional_metadata,
                    priority=priority,
                ),
            )

    async def aio_create(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
    ) -> V1WorkflowRunDetails:
        """
        Trigger a new workflow run.

        IMPORTANT: It's preferable to use `Workflow.run` (and similar) to trigger workflows if possible. This method is intended to be an escape hatch. For more details, see [the documentation](https://docs.hatchet.run/sdks/python/runnables#workflow).

        :param workflow_name: The name of the workflow to trigger.
        :param input: The input data for the workflow run.
        :param additional_metadata: Additional metadata associated with the workflow run.
        :param priority: The priority of the workflow run.

        :return: The details of the triggered workflow run.
        """
        return await asyncio.to_thread(
            self.create, workflow_name, input, additional_metadata, priority
        )

    def replay(self, run_id: str) -> None:
        """
        Replay a task or workflow run.

        :param run_id: The external ID of the task or workflow run to replay.
        :return: None
        """
        self.bulk_replay(opts=BulkCancelReplayOpts(ids=[run_id]))

    async def aio_replay(self, run_id: str) -> None:
        """
        Replay a task or workflow run.

        :param run_id: The external ID of the task or workflow run to replay.
        :return: None
        """
        return await asyncio.to_thread(self.replay, run_id)

    def bulk_replay(self, opts: BulkCancelReplayOpts) -> None:
        """
        Replay task or workflow runs in bulk, according to a set of filters.

        :param opts: Options for bulk replay, including filters and IDs.
        :return: None
        """
        with self.client() as client:
            self._ta(client).v1_task_replay(
                tenant=self.client_config.tenant_id,
                v1_replay_task_request=opts.to_request("replay"),
            )

    async def aio_bulk_replay(self, opts: BulkCancelReplayOpts) -> None:
        """
        Replay task or workflow runs in bulk, according to a set of filters.

        :param opts: Options for bulk replay, including filters and IDs.
        :return: None
        """
        return await asyncio.to_thread(self.bulk_replay, opts)

    def cancel(self, run_id: str) -> None:
        """
        Cancel a task or workflow run.

        :param run_id: The external ID of the task or workflow run to cancel.
        :return: None
        """
        self.bulk_cancel(opts=BulkCancelReplayOpts(ids=[run_id]))

    async def aio_cancel(self, run_id: str) -> None:
        """
        Cancel a task or workflow run.

        :param run_id: The external ID of the task or workflow run to cancel.
        :return: None
        """
        return await asyncio.to_thread(self.cancel, run_id)

    def bulk_cancel(self, opts: BulkCancelReplayOpts) -> None:
        """
        Cancel task or workflow runs in bulk, according to a set of filters.

        :param opts: Options for bulk cancel, including filters and IDs.
        :return: None
        """
        with self.client() as client:
            self._ta(client).v1_task_cancel(
                tenant=self.client_config.tenant_id,
                v1_cancel_task_request=opts.to_request("cancel"),
            )

    async def aio_bulk_cancel(self, opts: BulkCancelReplayOpts) -> None:
        """
        Cancel task or workflow runs in bulk, according to a set of filters.

        :param opts: Options for bulk cancel, including filters and IDs.
        :return: None
        """
        return await asyncio.to_thread(self.bulk_cancel, opts)

    @retry
    def get_result(self, run_id: str) -> JSONSerializableMapping:
        """
        Get the result of a workflow run by its external ID.

        :param run_id: The external ID of the workflow run to retrieve the result for.
        :return: The result of the workflow run.
        """
        details = self.get(run_id)

        return details.run.output

    async def aio_get_result(self, run_id: str) -> JSONSerializableMapping:
        """
        Get the result of a workflow run by its external ID.

        :param run_id: The external ID of the workflow run to retrieve the result for.
        :return: The result of the workflow run.
        """
        details = await asyncio.to_thread(self.get, run_id)

        return details.run.output

    def get_run_ref(self, workflow_run_id: str) -> "WorkflowRunRef":
        """
        Get a reference to a workflow run.

        :param workflow_run_id: The ID of the workflow run to get a reference to.
        :return: A reference to the specified workflow run.
        """
        from hatchet_sdk.workflow_run import WorkflowRunRef

        return WorkflowRunRef(
            workflow_run_id=workflow_run_id,
            workflow_run_event_listener=self.workflow_run_event_listener,
            workflow_run_listener=self.workflow_run_listener,
            runs_client=self,
        )

    async def subscribe_to_stream(
        self,
        workflow_run_id: str,
    ) -> AsyncIterator[str]:
        ref = self.get_run_ref(workflow_run_id=workflow_run_id)

        async for chunk in ref.stream():
            if chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:
                yield chunk.payload
