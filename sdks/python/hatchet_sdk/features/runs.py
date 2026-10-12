from __future__ import annotations

import asyncio
import time
from datetime import UTC, datetime, timedelta
from typing import TYPE_CHECKING, Literal, overload

from pydantic import BaseModel, model_validator

from hatchet_sdk.clients.listeners.run_event_listener import (
    RunEventListenerClient,
    TaskRunEventType,
)
from hatchet_sdk.clients.rest.models.v1_branch_durable_task_request import (
    V1BranchDurableTaskRequest,
)
from hatchet_sdk.clients.rest.models.v1_cancel_task_request import V1CancelTaskRequest
from hatchet_sdk.clients.rest.models.v1_replay_task_request import V1ReplayTaskRequest
from hatchet_sdk.clients.rest.models.v1_task_filter import V1TaskFilter
from hatchet_sdk.clients.rest.models.v1_trigger_workflow_run_request import (
    V1TriggerWorkflowRunRequest,
)
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import (
    BaseRestClient,
    maybe_additional_metadata_to_kv,
)
from hatchet_sdk.types.run_status import RunStatus
from hatchet_sdk.utils.aio import gather_max_concurrency
from hatchet_sdk.utils.datetimes import partition_date_range
from hatchet_sdk.utils.iterables import create_chunks

if TYPE_CHECKING:
    from collections.abc import AsyncIterator

    from hatchet_sdk.clients.admin import AdminClient, WorkflowRunDetail
    from hatchet_sdk.clients.listeners.workflow_listener import (
        PooledWorkflowRunListener,
    )
    from hatchet_sdk.clients.rest.api.task_api import TaskApi
    from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi
    from hatchet_sdk.clients.rest.api_client import ApiClient
    from hatchet_sdk.clients.rest.models.v1_branch_durable_task_response import (
        V1BranchDurableTaskResponse,
    )
    from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
    from hatchet_sdk.clients.rest.models.v1_workflow_run_details import (
        V1WorkflowRunDetails,
    )
    from hatchet_sdk.config import ClientConfig
    from hatchet_sdk.types.priority import Priority
    from hatchet_sdk.utils.typing import JSONSerializableMapping
    from hatchet_sdk.workflow_run import WorkflowRunRef


class RunFilter(BaseModel):
    since: datetime
    until: datetime | None = None
    statuses: list[RunStatus] | None = None
    workflow_ids: list[str] | None = None
    additional_metadata: dict[str, str] | None = None


class BulkCancelReplayOptions(BaseModel):
    ids: list[str] | None = None
    filters: RunFilter | None = None

    @model_validator(mode="after")
    def validate_model(self) -> BulkCancelReplayOptions:
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


LIST_RUNS_PAGE_SIZE = 100


def _max_rows_needed_per_date_range(
    offset: int | None, limit: int | None
) -> int | None:
    if limit is None:
        return None

    return (offset or 0) + limit


def _merge_runs_across_date_ranges(
    runs_per_date_range: list[list[V1TaskSummary]],
    offset: int | None,
    limit: int | None,
) -> list[V1TaskSummary]:
    ## a run can be returned for two adjacent date ranges
    run_id_to_run = {
        run.metadata.id: run for runs in runs_per_date_range for run in runs
    }

    newest_first = sorted(
        run_id_to_run.values(),
        key=lambda run: run.created_at,
        reverse=True,
    )

    start = offset or 0
    end = None if limit is None else start + limit

    return newest_first[start:end]


class RunsClient(BaseRestClient):
    """
    The runs client is a client for interacting with task and workflow runs within Hatchet.
    """

    def __init__(
        self,
        config: ClientConfig,
        workflow_run_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
        admin_client: AdminClient,
    ) -> None:
        super().__init__(config)

        self._workflow_run_listener = workflow_run_listener
        self._workflow_run_event_listener = workflow_run_event_listener
        self._admin_client = admin_client

    def _wra(self, client: ApiClient) -> WorkflowRunsApi:
        from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi

        return WorkflowRunsApi(client)

    def _ta(self, client: ApiClient) -> TaskApi:
        from hatchet_sdk.clients.rest.api.task_api import TaskApi

        return TaskApi(client)

    def get_task_run(self, task_run_id: str) -> V1TaskSummary:
        """
        Get task run details for a given task run ID.

        :param task_run_id: The ID of the task run to retrieve details for.
        :return: Task run details for the specified task run ID.
        """
        with self.client() as client:
            v1_task_get = tenacity_retry(
                self._ta(client).v1_task_get, self.client_config.tenacity
            )
            return v1_task_get(task_run_id)

    async def aio_get_task_run(self, task_run_id: str) -> V1TaskSummary:
        """
        Get task run details for a given task run ID.

        :param task_run_id: The ID of the task run to retrieve details for.
        :return: Task run details for the specified task run ID.
        """
        return await asyncio.to_thread(self.get_task_run, task_run_id)

    def get(self, workflow_run_id: str) -> V1WorkflowRunDetails:
        """
        Get workflow run details for a given workflow run ID.

        :param workflow_run_id: The ID of the workflow run to retrieve details for.
        :return: Workflow run details for the specified workflow run ID.
        """
        with self.client() as client:
            v1_workflow_run_get = tenacity_retry(
                self._wra(client).v1_workflow_run_get, self.client_config.tenacity
            )
            return v1_workflow_run_get(str(workflow_run_id))

    async def aio_get(self, workflow_run_id: str) -> V1WorkflowRunDetails:
        """
        Get workflow run details for a given workflow run ID.

        :param workflow_run_id: The ID of the workflow run to retrieve details for.
        :return: Workflow run details for the specified workflow run ID.
        """
        return await asyncio.to_thread(self.get, workflow_run_id)

    def get_status(self, workflow_run_id: str) -> RunStatus:
        """
        Get workflow run status for a given workflow run ID.

        :param workflow_run_id: The ID of the workflow run to retrieve details for.
        :return: The task status
        """
        details = self.get_details(workflow_run_id)
        return details.status

    async def aio_get_status(self, workflow_run_id: str) -> RunStatus:
        """
        Get workflow run status for a given workflow run ID.

        :param workflow_run_id: The ID of the workflow run to retrieve details for.
        :return: The task status
        """
        return await asyncio.to_thread(self.get_status, workflow_run_id)

    def _perform_action_with_pagination(
        self,
        action: Literal["cancel", "replay"],
        statuses: list[RunStatus],
        sleep_time: int = 3,
        chunk_size: int = 500,
        since: datetime | None = None,
        until: datetime | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
    ) -> None:
        """
        Perform a bulk action (cancel or replay) on runs matching the specified filters in chunks.

        The motivation for this method is to provide an easy way to perform bulk operations by filters over a larger number of runs than
        the API would normally be able to handle, with automatic pagination and chunking to help limit the pressure on the API.

        This method first pulls the IDs of the runs from the API, and then feeds them back to the API in chunks.

        :param action: The action to perform, either "cancel" or "replay".
        :param statuses: The statuses to filter runs by.
        :param sleep_time: The time to sleep between processing chunks, in seconds.
        :param chunk_size: The maximum number of run IDs to process in each chunk.
        :param since: The start time for filtering runs.
        :param until: The end time for filtering runs.
        :param additional_metadata: Additional metadata to filter runs by.
        :param workflow_ids: The workflow IDs to filter runs by.
        """
        until = until or datetime.now(tz=UTC)
        since = since or (until - timedelta(days=1))

        with self.client() as client:
            external_ids = self._wra(client).v1_workflow_run_external_ids_list(
                tenant=self.client_config.tenant_id,
                since=since,
                until=until,
                additional_metadata=maybe_additional_metadata_to_kv(
                    additional_metadata
                ),
                statuses=statuses,
                workflow_ids=workflow_ids,
            )

        chunks = list(create_chunks(external_ids, chunk_size))
        func = self.bulk_cancel if action == "cancel" else self.bulk_replay

        for ix, chunk in enumerate(chunks):
            self.client_config.logger.info(
                f"processing chunk {ix + 1}/{len(chunks)} of {len(chunk)} ids"  # noqa: G004
            )

            options = BulkCancelReplayOptions(ids=chunk)
            func(options=options)

            time.sleep(sleep_time)

    def bulk_replay_by_filters_with_pagination(
        self,
        sleep_time: int = 3,
        chunk_size: int = 500,
        since: datetime | None = None,
        until: datetime | None = None,
        statuses: list[RunStatus] | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
    ) -> None:
        """
        Replay runs matching the specified filters in chunks.

        The motivation for this method is to provide an easy way to perform bulk operations by filters over a larger number of runs than
        the API would normally be able to handle, with automatic pagination and chunking to help limit the pressure on the API.

        This method first pulls the IDs of the runs from the API, and then feeds them back to the API in chunks.

        :param sleep_time: The time to sleep between processing chunks, in seconds.
        :param chunk_size: The maximum number of run IDs to process in each chunk.
        :param since: The start time for filtering runs.
        :param until: The end time for filtering runs.
        :param statuses: The statuses to filter runs by.
        :param additional_metadata: Additional metadata to filter runs by.
        :param workflow_ids: The workflow IDs to filter runs by.
        """

        self._perform_action_with_pagination(
            since=since,
            action="replay",
            sleep_time=sleep_time,
            chunk_size=chunk_size,
            until=until,
            statuses=statuses or [RunStatus.FAILED, RunStatus.CANCELLED],
            additional_metadata=additional_metadata,
            workflow_ids=workflow_ids,
        )

    def bulk_cancel_by_filters_with_pagination(
        self,
        sleep_time: int = 3,
        chunk_size: int = 500,
        since: datetime | None = None,
        until: datetime | None = None,
        statuses: list[RunStatus] | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
    ) -> None:
        """
        Cancel runs matching the specified filters in chunks.

        The motivation for this method is to provide an easy way to perform bulk operations by filters over a larger number of runs than
        the API would normally be able to handle, with automatic pagination and chunking to help limit the pressure on the API.

        This method first pulls the IDs of the runs from the API, and then feeds them back to the API in chunks.

        :param sleep_time: The time to sleep between processing chunks, in seconds.
        :param chunk_size: The maximum number of run IDs to process in each chunk.
        :param since: The start time for filtering runs.
        :param until: The end time for filtering runs.
        :param statuses: The statuses to filter runs by.
        :param additional_metadata: Additional metadata to filter runs by.
        :param workflow_ids: The workflow IDs to filter runs by.
        """

        self._perform_action_with_pagination(
            since=since,
            action="cancel",
            sleep_time=sleep_time,
            chunk_size=chunk_size,
            until=until,
            statuses=statuses or [RunStatus.RUNNING, RunStatus.QUEUED],
            additional_metadata=additional_metadata,
            workflow_ids=workflow_ids,
        )

    async def aio_bulk_replay_by_filters_with_pagination(
        self,
        sleep_time: int = 3,
        chunk_size: int = 500,
        since: datetime | None = None,
        until: datetime | None = None,
        statuses: list[RunStatus] | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
    ) -> None:
        """
        Replay runs matching the specified filters in chunks.

        The motivation for this method is to provide an easy way to perform bulk operations by filters over a larger number of runs than
        the API would normally be able to handle, with automatic pagination and chunking to help limit the pressure on the API.

        This method first pulls the IDs of the runs from the API, and then feeds them back to the API in chunks.

        :param sleep_time: The time to sleep between processing chunks, in seconds.
        :param chunk_size: The maximum number of run IDs to process in each chunk.
        :param since: The start time for filtering runs.
        :param until: The end time for filtering runs.
        :param statuses: The statuses to filter runs by.
        :param additional_metadata: Additional metadata to filter runs by.
        :param workflow_ids: The workflow IDs to filter runs by.
        """

        await asyncio.to_thread(
            self._perform_action_with_pagination,
            since=since,
            action="replay",
            sleep_time=sleep_time,
            chunk_size=chunk_size,
            until=until,
            statuses=statuses or [RunStatus.FAILED, RunStatus.CANCELLED],
            additional_metadata=additional_metadata,
            workflow_ids=workflow_ids,
        )

    async def aio_bulk_cancel_by_filters_with_pagination(
        self,
        sleep_time: int = 3,
        chunk_size: int = 500,
        since: datetime | None = None,
        until: datetime | None = None,
        statuses: list[RunStatus] | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
    ) -> None:
        """
        Cancel runs matching the specified filters in chunks.

        The motivation for this method is to provide an easy way to perform bulk operations by filters over a larger number of runs than
        the API would normally be able to handle, with automatic pagination and chunking to help limit the pressure on the API.

        This method first pulls the IDs of the runs from the API, and then feeds them back to the API in chunks.

        :param sleep_time: The time to sleep between processing chunks, in seconds.
        :param chunk_size: The maximum number of run IDs to process in each chunk.
        :param since: The start time for filtering runs.
        :param until: The end time for filtering runs.
        :param statuses: The statuses to filter runs by.
        :param additional_metadata: Additional metadata to filter runs by.
        :param workflow_ids: The workflow IDs to filter runs by.
        """

        await asyncio.to_thread(
            self._perform_action_with_pagination,
            since=since,
            action="cancel",
            sleep_time=sleep_time,
            chunk_size=chunk_size,
            until=until,
            statuses=statuses or [RunStatus.RUNNING, RunStatus.QUEUED],
            additional_metadata=additional_metadata,
            workflow_ids=workflow_ids,
        )

    def _list_runs_in_date_range(
        self,
        since: datetime,
        until: datetime,
        max_rows: int | None,
        only_tasks: bool,
        statuses: list[RunStatus] | None,
        additional_metadata: dict[str, str] | None,
        workflow_ids: list[str] | None,
        worker_id: str | None,
        parent_task_external_id: str | None,
        triggering_event_external_id: str | None,
        include_payloads: bool,
    ) -> list[V1TaskSummary]:
        with self.client() as client:
            v1_workflow_run_list = tenacity_retry(
                self._wra(client).v1_workflow_run_list, self.client_config.tenacity
            )

            runs: list[V1TaskSummary] = []

            while max_rows is None or len(runs) < max_rows:
                page = v1_workflow_run_list(
                    tenant=self.client_config.tenant_id,
                    since=since,
                    until=until,
                    only_tasks=only_tasks,
                    offset=len(runs),
                    limit=LIST_RUNS_PAGE_SIZE,
                    statuses=statuses,
                    additional_metadata=maybe_additional_metadata_to_kv(
                        additional_metadata
                    ),
                    workflow_ids=workflow_ids,
                    worker_id=worker_id,
                    parent_task_external_id=parent_task_external_id,
                    triggering_event_external_id=triggering_event_external_id,
                    include_payloads=include_payloads,
                ).rows

                runs.extend(page)

                if len(page) < LIST_RUNS_PAGE_SIZE:
                    break

            return runs

    async def aio_list(
        self,
        since: datetime | None = None,
        only_tasks: bool = False,
        offset: int | None = None,
        limit: int | None = None,
        statuses: list[RunStatus] | None = None,
        until: datetime | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
        worker_id: str | None = None,
        parent_task_external_id: str | None = None,
        triggering_event_external_id: str | None = None,
        include_payloads: bool = True,
    ) -> list[V1TaskSummary]:
        """
        List task runs according to a set of filters, newest first.

        :param since: The start time for filtering task runs.
        :param only_tasks: Whether to only list task runs.
        :param offset: The number of task runs to skip, counted across the whole time range.
        :param limit: The maximum number of task runs to return, counted across the whole time range. Returns every matching task run if not set.
        :param statuses: The statuses to filter task runs by.
        :param until: The end time for filtering task runs.
        :param additional_metadata: Additional metadata to filter task runs by.
        :param workflow_ids: The workflow IDs to filter task runs by.
        :param worker_id: The worker ID to filter task runs by.
        :param parent_task_external_id: The parent task external ID to filter task runs by.
        :param triggering_event_external_id: The event id that triggered the task run.
        :param include_payloads: Whether to include payloads in the response.

        :return: A list of task runs matching the specified filters.
        """

        date_ranges = partition_date_range(
            since=since or datetime.now(tz=UTC) - timedelta(days=1),
            until=until or datetime.now(tz=UTC),
        )

        runs_per_date_range = await gather_max_concurrency(
            *[
                asyncio.to_thread(
                    self._list_runs_in_date_range,
                    since=s,
                    until=u,
                    max_rows=_max_rows_needed_per_date_range(offset, limit),
                    only_tasks=only_tasks,
                    statuses=statuses,
                    additional_metadata=additional_metadata,
                    workflow_ids=workflow_ids,
                    worker_id=worker_id,
                    parent_task_external_id=parent_task_external_id,
                    triggering_event_external_id=triggering_event_external_id,
                    include_payloads=include_payloads,
                )
                for s, u in date_ranges
            ],
            max_concurrency=3,
        )

        return _merge_runs_across_date_ranges(runs_per_date_range, offset, limit)

    def list(
        self,
        since: datetime | None = None,
        only_tasks: bool = False,
        offset: int | None = None,
        limit: int | None = None,
        statuses: list[RunStatus] | None = None,
        until: datetime | None = None,
        additional_metadata: dict[str, str] | None = None,
        workflow_ids: list[str] | None = None,
        worker_id: str | None = None,
        parent_task_external_id: str | None = None,
        triggering_event_external_id: str | None = None,
        include_payloads: bool = True,
    ) -> list[V1TaskSummary]:
        """
        List task runs according to a set of filters, newest first.

        :param since: The start time for filtering task runs.
        :param only_tasks: Whether to only list task runs.
        :param offset: The number of task runs to skip, counted across the whole time range.
        :param limit: The maximum number of task runs to return, counted across the whole time range. Returns every matching task run if not set.
        :param statuses: The statuses to filter task runs by.
        :param until: The end time for filtering task runs.
        :param additional_metadata: Additional metadata to filter task runs by.
        :param workflow_ids: The workflow IDs to filter task runs by.
        :param worker_id: The worker ID to filter task runs by.
        :param parent_task_external_id: The parent task external ID to filter task runs by.
        :param triggering_event_external_id: The event id that triggered the task run.
        :param include_payloads: Whether to include payloads in the response.

        :return: A list of task runs matching the specified filters.
        """

        date_ranges = partition_date_range(
            since=since or datetime.now(tz=UTC) - timedelta(days=1),
            until=until or datetime.now(tz=UTC),
        )

        runs_per_date_range = [
            self._list_runs_in_date_range(
                since=s,
                until=u,
                max_rows=_max_rows_needed_per_date_range(offset, limit),
                only_tasks=only_tasks,
                statuses=statuses,
                additional_metadata=additional_metadata,
                workflow_ids=workflow_ids,
                worker_id=worker_id,
                parent_task_external_id=parent_task_external_id,
                triggering_event_external_id=triggering_event_external_id,
                include_payloads=include_payloads,
            )
            for s, u in date_ranges
        ]

        return _merge_runs_across_date_ranges(runs_per_date_range, offset, limit)

    def create(
        self,
        workflow_name: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: Priority | None = None,
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
        priority: Priority | None = None,
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
        self.bulk_replay(options=BulkCancelReplayOptions(ids=[run_id]))

    async def aio_replay(self, run_id: str) -> None:
        """
        Replay a task or workflow run.

        :param run_id: The external ID of the task or workflow run to replay.
        :return: None
        """
        return await asyncio.to_thread(self.replay, run_id)

    def bulk_replay(self, options: BulkCancelReplayOptions) -> None:
        """
        Replay task or workflow runs in bulk, according to a set of filters.

        :param options: Options for bulk replay, including filters and IDs.
        :return: None
        """
        with self.client() as client:
            self._ta(client).v1_task_replay(
                tenant=self.client_config.tenant_id,
                v1_replay_task_request=options.to_request("replay"),
            )

    async def aio_bulk_replay(self, options: BulkCancelReplayOptions) -> None:
        """
        Replay task or workflow runs in bulk, according to a set of filters.

        :param options: Options for bulk replay, including filters and IDs.
        :return: None
        """
        return await asyncio.to_thread(self.bulk_replay, options)

    def cancel(self, run_id: str) -> None:
        """
        Cancel a task or workflow run.

        :param run_id: The external ID of the task or workflow run to cancel.
        :return: None
        """
        self.bulk_cancel(options=BulkCancelReplayOptions(ids=[run_id]))

    async def aio_cancel(self, run_id: str) -> None:
        """
        Cancel a task or workflow run.

        :param run_id: The external ID of the task or workflow run to cancel.
        :return: None
        """
        return await asyncio.to_thread(self.cancel, run_id)

    def bulk_cancel(self, options: BulkCancelReplayOptions) -> None:
        """
        Cancel task or workflow runs in bulk, according to a set of filters.

        :param options: Options for bulk cancel, including filters and IDs.
        :return: None
        """
        with self.client() as client:
            self._ta(client).v1_task_cancel(
                tenant=self.client_config.tenant_id,
                v1_cancel_task_request=options.to_request("cancel"),
            )

    async def aio_bulk_cancel(self, options: BulkCancelReplayOptions) -> None:
        """
        Cancel task or workflow runs in bulk, according to a set of filters.

        :param options: Options for bulk cancel, including filters and IDs.
        :return: None
        """
        return await asyncio.to_thread(self.bulk_cancel, options)

    def get_result(self, workflow_run_id: str) -> JSONSerializableMapping:
        """
        Get the result of a workflow run by its external ID.

        :param workflow_run_id: The external ID of the workflow run to retrieve the result for.
        :return: The result of the workflow run.
        """
        details = self.get(workflow_run_id)

        return details.run.output

    async def aio_get_result(self, workflow_run_id: str) -> JSONSerializableMapping:
        """
        Get the result of a workflow run by its external ID.

        :param workflow_run_id: The external ID of the workflow run to retrieve the result for.
        :return: The result of the workflow run.
        """
        details = await asyncio.to_thread(self.get, workflow_run_id)

        return details.run.output

    def get_run_ref(self, workflow_run_id: str) -> WorkflowRunRef:
        """
        Get a reference to a workflow run.

        :param workflow_run_id: The ID of the workflow run to get a reference to.
        :return: A reference to the specified workflow run.
        """
        from hatchet_sdk.workflow_run import WorkflowRunRef

        return WorkflowRunRef(
            workflow_run_id=workflow_run_id,
            workflow_run_event_listener=self._workflow_run_event_listener,
            workflow_run_listener=self._workflow_run_listener,
            admin_client=self._admin_client,
        )

    async def aio_subscribe_to_stream(
        self,
        workflow_run_id: str,
    ) -> AsyncIterator[str]:
        """
        Subscribe to the stream chunks sent by a run via `ctx.put_stream`.

        :param workflow_run_id: The ID of the workflow run to subscribe to.
        :yield: Each stream chunk as it arrives.
        :ytype: str
        """
        ref = self.get_run_ref(workflow_run_id=workflow_run_id)

        async for chunk in ref._stream():
            if chunk.type == TaskRunEventType.STREAM:
                yield chunk.payload

    def reset_durable_task(
        self, task_run_id: str, node_id: int, branch_id: int
    ) -> V1BranchDurableTaskResponse:
        """
        Reset a durable task from a specific node id, creating a new branch.

        :param task_run_id: The external ID (UUID) of the durable task run to reset.
        :param node_id: The node ID to replay from.
        :param branch_id: The branch ID to replay from.
        :return: The reset response containing the new node_id and branch_id.
        """
        with self.client() as client:
            return self._wra(client).v1_durable_task_branch(
                tenant=self.client_config.tenant_id,
                v1_branch_durable_task_request=V1BranchDurableTaskRequest(
                    taskExternalId=task_run_id, nodeId=node_id, branchId=branch_id
                ),
            )

    async def aio_reset_durable_task(
        self, task_run_id: str, node_id: int, branch_id: int
    ) -> V1BranchDurableTaskResponse:
        """
        Reset a durable task from a specific node id, creating a new branch.

        :param task_run_id: The external ID (UUID) of the durable task run to reset.
        :param node_id: The node ID to replay from.
        :param branch_id: The branch ID to replay from.
        :return: The reset response containing the new node_id and branch_id.
        """
        return await asyncio.to_thread(
            self.reset_durable_task, task_run_id, node_id, branch_id
        )

    def get_details(self, workflow_run_id: str) -> WorkflowRunDetail:
        return self._admin_client.get_details(workflow_run_id=workflow_run_id)

    async def aio_get_details(self, workflow_run_id: str) -> WorkflowRunDetail:
        return await asyncio.to_thread(self._admin_client.get_details, workflow_run_id)
