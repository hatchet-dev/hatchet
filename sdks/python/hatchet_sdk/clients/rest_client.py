import asyncio
import atexit
import datetime
import threading
from typing import Coroutine, TypeVar

from pydantic import StrictInt

from hatchet_sdk.clients.rest.api.event_api import EventApi
from hatchet_sdk.clients.rest.api.log_api import LogApi
from hatchet_sdk.clients.rest.api.step_run_api import StepRunApi
from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi
from hatchet_sdk.clients.rest.api.workflow_run_api import WorkflowRunApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.configuration import Configuration
from hatchet_sdk.clients.rest.models.create_cron_workflow_trigger_request import (
    CreateCronWorkflowTriggerRequest,
)
from hatchet_sdk.clients.rest.models.cron_workflows import CronWorkflows
from hatchet_sdk.clients.rest.models.cron_workflows_list import CronWorkflowsList
from hatchet_sdk.clients.rest.models.cron_workflows_order_by_field import (
    CronWorkflowsOrderByField,
)
from hatchet_sdk.clients.rest.models.event_list import EventList
from hatchet_sdk.clients.rest.models.event_order_by_direction import (
    EventOrderByDirection,
)
from hatchet_sdk.clients.rest.models.event_order_by_field import EventOrderByField
from hatchet_sdk.clients.rest.models.event_update_cancel200_response import (
    EventUpdateCancel200Response,
)
from hatchet_sdk.clients.rest.models.log_line_level import LogLineLevel
from hatchet_sdk.clients.rest.models.log_line_list import LogLineList
from hatchet_sdk.clients.rest.models.log_line_order_by_direction import (
    LogLineOrderByDirection,
)
from hatchet_sdk.clients.rest.models.log_line_order_by_field import LogLineOrderByField
from hatchet_sdk.clients.rest.models.replay_event_request import ReplayEventRequest
from hatchet_sdk.clients.rest.models.replay_workflow_runs_request import (
    ReplayWorkflowRunsRequest,
)
from hatchet_sdk.clients.rest.models.replay_workflow_runs_response import (
    ReplayWorkflowRunsResponse,
)
from hatchet_sdk.clients.rest.models.schedule_workflow_run_request import (
    ScheduleWorkflowRunRequest,
)
from hatchet_sdk.clients.rest.models.scheduled_workflows import ScheduledWorkflows
from hatchet_sdk.clients.rest.models.scheduled_workflows_list import (
    ScheduledWorkflowsList,
)
from hatchet_sdk.clients.rest.models.scheduled_workflows_order_by_field import (
    ScheduledWorkflowsOrderByField,
)
from hatchet_sdk.clients.rest.models.trigger_workflow_run_request import (
    TriggerWorkflowRunRequest,
)
from hatchet_sdk.clients.rest.models.workflow import Workflow
from hatchet_sdk.clients.rest.models.workflow_kind import WorkflowKind
from hatchet_sdk.clients.rest.models.workflow_list import WorkflowList
from hatchet_sdk.clients.rest.models.workflow_run import WorkflowRun
from hatchet_sdk.clients.rest.models.workflow_run_list import WorkflowRunList
from hatchet_sdk.clients.rest.models.workflow_run_order_by_direction import (
    WorkflowRunOrderByDirection,
)
from hatchet_sdk.clients.rest.models.workflow_run_order_by_field import (
    WorkflowRunOrderByField,
)
from hatchet_sdk.clients.rest.models.workflow_run_status import WorkflowRunStatus
from hatchet_sdk.clients.rest.models.workflow_runs_cancel_request import (
    WorkflowRunsCancelRequest,
)
from hatchet_sdk.clients.rest.models.workflow_version import WorkflowVersion
from hatchet_sdk.utils.types import JSONSerializableDict

## Type variables to use with coroutines.
## See https://stackoverflow.com/questions/73240620/the-right-way-to-type-hint-a-coroutine-function
## Return type
R = TypeVar("R")

## Yield type
Y = TypeVar("Y")

## Send type
S = TypeVar("S")


class RestApi:
    def __init__(self, host: str, api_key: str, tenant_id: str):
        self.tenant_id = tenant_id

        self.config = Configuration(
            host=host,
            access_token=api_key,
        )

        self.api_client = ApiClient(configuration=self.config)

        self.workflow_api = WorkflowApi(self.api_client)
        self.workflow_run_api = WorkflowRunApi(self.api_client)
        self.step_run_api = StepRunApi(self.api_client)
        self.event_api = EventApi(self.api_client)
        self.log_api = LogApi(self.api_client)

        self._loop = asyncio.new_event_loop()
        self._thread = threading.Thread(target=self._run_event_loop, daemon=True)
        self._thread.start()

        # Register the cleanup method to be called on exit
        atexit.register(self._cleanup)

    async def close(self) -> None:
        # Ensure the aiohttp client session is closed
        await self.api_client.close()

    async def aio_list_workflows(self) -> WorkflowList:
        return await self.workflow_api.workflow_list(
            tenant=self.tenant_id,
        )

    async def aio_get_workflow(self, workflow_id: str) -> Workflow:
        return await self.workflow_api.workflow_get(
            workflow=workflow_id,
        )

    async def aio_get_workflow_version(
        self, workflow_id: str, version: str | None = None
    ) -> WorkflowVersion:
        return await self.workflow_api.workflow_version_get(
            workflow=workflow_id,
            version=version,
        )

    async def aio_list_workflow_runs(
        self,
        workflow_id: str | None = None,
        offset: int | None = None,
        limit: int | None = None,
        event_id: str | None = None,
        parent_workflow_run_id: str | None = None,
        parent_step_run_id: str | None = None,
        statuses: list[WorkflowRunStatus] | None = None,
        kinds: list[WorkflowKind] | None = None,
        additional_metadata: list[str] | None = None,
        order_by_field: WorkflowRunOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> WorkflowRunList:
        return await self.workflow_api.workflow_run_list(
            tenant=self.tenant_id,
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            event_id=event_id,
            parent_workflow_run_id=parent_workflow_run_id,
            parent_step_run_id=parent_step_run_id,
            statuses=statuses,
            kinds=kinds,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
        )

    async def aio_get_workflow_run(self, workflow_run_id: str) -> WorkflowRun:
        return await self.workflow_api.workflow_run_get(
            tenant=self.tenant_id,
            workflow_run=workflow_run_id,
        )

    async def aio_replay_workflow_run(
        self, workflow_run_ids: list[str]
    ) -> ReplayWorkflowRunsResponse:
        return await self.workflow_run_api.workflow_run_update_replay(
            tenant=self.tenant_id,
            replay_workflow_runs_request=ReplayWorkflowRunsRequest(
                workflowRunIds=workflow_run_ids,
            ),
        )

    async def aio_cancel_workflow_run(
        self, workflow_run_id: str
    ) -> EventUpdateCancel200Response:
        return await self.workflow_run_api.workflow_run_cancel(
            tenant=self.tenant_id,
            workflow_runs_cancel_request=WorkflowRunsCancelRequest(
                workflowRunIds=[workflow_run_id],
            ),
        )

    async def aio_bulk_cancel_workflow_runs(
        self, workflow_run_ids: list[str]
    ) -> EventUpdateCancel200Response:
        return await self.workflow_run_api.workflow_run_cancel(
            tenant=self.tenant_id,
            workflow_runs_cancel_request=WorkflowRunsCancelRequest(
                workflowRunIds=workflow_run_ids,
            ),
        )

    async def aio_create_workflow_run(
        self,
        workflow_id: str,
        input: JSONSerializableDict,
        version: str | None = None,
        additional_metadata: JSONSerializableDict = {},
    ) -> WorkflowRun:
        return await self.workflow_run_api.workflow_run_create(
            workflow=workflow_id,
            version=version,
            trigger_workflow_run_request=TriggerWorkflowRunRequest(
                input=input,
                additionalMetadata=additional_metadata,
            ),
        )

    async def aio_create_cron(
        self,
        workflow_name: str,
        cron_name: str,
        expression: str,
        input: JSONSerializableDict,
        additional_metadata: JSONSerializableDict,
    ) -> CronWorkflows:
        return await self.workflow_run_api.cron_workflow_trigger_create(
            tenant=self.tenant_id,
            workflow=workflow_name,
            create_cron_workflow_trigger_request=CreateCronWorkflowTriggerRequest(
                cronName=cron_name,
                cronExpression=expression,
                input=input,
                additionalMetadata=additional_metadata,
            ),
        )

    async def aio_delete_cron(self, cron_trigger_id: str) -> None:
        await self.workflow_api.workflow_cron_delete(
            tenant=self.tenant_id,
            cron_workflow=cron_trigger_id,
        )

    async def aio_list_crons(
        self,
        offset: StrictInt | None = None,
        limit: StrictInt | None = None,
        workflow_id: str | None = None,
        additional_metadata: list[str] | None = None,
        order_by_field: CronWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> CronWorkflowsList:
        return await self.workflow_api.cron_workflow_list(
            tenant=self.tenant_id,
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
        )

    async def aio_get_cron(self, cron_trigger_id: str) -> CronWorkflows:
        return await self.workflow_api.workflow_cron_get(
            tenant=self.tenant_id,
            cron_workflow=cron_trigger_id,
        )

    async def aio_create_schedule(
        self,
        name: str,
        trigger_at: datetime.datetime,
        input: JSONSerializableDict,
        additional_metadata: JSONSerializableDict,
    ) -> ScheduledWorkflows:
        return await self.workflow_run_api.scheduled_workflow_run_create(
            tenant=self.tenant_id,
            workflow=name,
            schedule_workflow_run_request=ScheduleWorkflowRunRequest(
                triggerAt=trigger_at,
                input=input,
                additionalMetadata=additional_metadata,
            ),
        )

    async def aio_delete_schedule(self, scheduled_trigger_id: str) -> None:
        await self.workflow_api.workflow_scheduled_delete(
            tenant=self.tenant_id,
            scheduled_workflow_run=scheduled_trigger_id,
        )

    async def aio_list_schedule(
        self,
        offset: StrictInt | None = None,
        limit: StrictInt | None = None,
        workflow_id: str | None = None,
        additional_metadata: list[str] | None = None,
        parent_workflow_run_id: str | None = None,
        parent_step_run_id: str | None = None,
        order_by_field: ScheduledWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> ScheduledWorkflowsList:
        return await self.workflow_api.workflow_scheduled_list(
            tenant=self.tenant_id,
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            parent_workflow_run_id=parent_workflow_run_id,
            parent_step_run_id=parent_step_run_id,
            additional_metadata=additional_metadata,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
        )

    async def aio_get_schedule(self, scheduled_trigger_id: str) -> ScheduledWorkflows:
        return await self.workflow_api.workflow_scheduled_get(
            tenant=self.tenant_id,
            scheduled_workflow_run=scheduled_trigger_id,
        )

    async def aio_list_logs(
        self,
        step_run_id: str,
        offset: int | None = None,
        limit: int | None = None,
        levels: list[LogLineLevel] | None = None,
        search: str | None = None,
        order_by_field: LogLineOrderByField | None = None,
        order_by_direction: LogLineOrderByDirection | None = None,
    ) -> LogLineList:
        return await self.log_api.log_line_list(
            step_run=step_run_id,
            offset=offset,
            limit=limit,
            levels=levels,
            search=search,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
        )

    async def aio_list_events(
        self,
        offset: int | None = None,
        limit: int | None = None,
        keys: list[str] | None = None,
        workflows: list[str] | None = None,
        statuses: list[WorkflowRunStatus] | None = None,
        search: str | None = None,
        order_by_field: EventOrderByField | None = None,
        order_by_direction: EventOrderByDirection | None = None,
        additional_metadata: list[str] | None = None,
    ) -> EventList:
        return await self.event_api.event_list(
            tenant=self.tenant_id,
            offset=offset,
            limit=limit,
            keys=keys,
            workflows=workflows,
            statuses=statuses,
            search=search,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
            additional_metadata=additional_metadata,
        )

    async def aio_replay_events(self, event_ids: list[str] | EventList) -> EventList:
        if isinstance(event_ids, EventList):
            rows = event_ids.rows or []
            event_ids = [r.metadata.id for r in rows]

        return await self.event_api.event_update_replay(
            tenant=self.tenant_id,
            replay_event_request=ReplayEventRequest(eventIds=event_ids),
        )

    def _cleanup(self) -> None:
        """
        Stop the running thread and clean up the event loop.
        """
        self._run_coroutine(self.close())
        self._loop.call_soon_threadsafe(self._loop.stop)
        self._thread.join()

    def _run_event_loop(self) -> None:
        """
        Run the asyncio event loop in a separate thread.
        """
        asyncio.set_event_loop(self._loop)
        self._loop.run_forever()

    def _run_coroutine(self, coro: Coroutine[Y, S, R]) -> R:
        """
        Execute a coroutine in the event loop and return the result.
        """
        future = asyncio.run_coroutine_threadsafe(coro, self._loop)
        return future.result()

    def workflow_list(self) -> WorkflowList:
        return self._run_coroutine(self.aio_list_workflows())

    def workflow_get(self, workflow_id: str) -> Workflow:
        return self._run_coroutine(self.aio_get_workflow(workflow_id))

    def workflow_version_get(
        self, workflow_id: str, version: str | None = None
    ) -> WorkflowVersion:
        return self._run_coroutine(self.aio_get_workflow_version(workflow_id, version))

    def workflow_run_list(
        self,
        workflow_id: str | None = None,
        offset: int | None = None,
        limit: int | None = None,
        event_id: str | None = None,
        parent_workflow_run_id: str | None = None,
        parent_step_run_id: str | None = None,
        statuses: list[WorkflowRunStatus] | None = None,
        kinds: list[WorkflowKind] | None = None,
        additional_metadata: list[str] | None = None,
        order_by_field: WorkflowRunOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> WorkflowRunList:
        return self._run_coroutine(
            self.aio_list_workflow_runs(
                workflow_id=workflow_id,
                offset=offset,
                limit=limit,
                event_id=event_id,
                parent_workflow_run_id=parent_workflow_run_id,
                parent_step_run_id=parent_step_run_id,
                statuses=statuses,
                kinds=kinds,
                additional_metadata=additional_metadata,
                order_by_field=order_by_field,
                order_by_direction=order_by_direction,
            )
        )

    def workflow_run_get(self, workflow_run_id: str) -> WorkflowRun:
        return self._run_coroutine(self.aio_get_workflow_run(workflow_run_id))

    def workflow_run_cancel(self, workflow_run_id: str) -> EventUpdateCancel200Response:
        return self._run_coroutine(self.aio_cancel_workflow_run(workflow_run_id))

    def workflow_run_bulk_cancel(
        self, workflow_run_ids: list[str]
    ) -> EventUpdateCancel200Response:
        return self._run_coroutine(self.aio_bulk_cancel_workflow_runs(workflow_run_ids))

    def workflow_run_create(
        self,
        workflow_id: str,
        input: JSONSerializableDict,
        version: str | None = None,
        additional_metadata: JSONSerializableDict = {},
    ) -> WorkflowRun:
        return self._run_coroutine(
            self.aio_create_workflow_run(
                workflow_id, input, version, additional_metadata
            )
        )

    def cron_create(
        self,
        workflow_name: str,
        cron_name: str,
        expression: str,
        input: JSONSerializableDict,
        additional_metadata: JSONSerializableDict,
    ) -> CronWorkflows:
        return self._run_coroutine(
            self.aio_create_cron(
                workflow_name, cron_name, expression, input, additional_metadata
            )
        )

    def cron_delete(self, cron_trigger_id: str) -> None:
        self._run_coroutine(self.aio_delete_cron(cron_trigger_id))

    def cron_list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        additional_metadata: list[str] | None = None,
        order_by_field: CronWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> CronWorkflowsList:
        return self._run_coroutine(
            self.aio_list_crons(
                offset,
                limit,
                workflow_id,
                additional_metadata,
                order_by_field,
                order_by_direction,
            )
        )

    def cron_get(self, cron_trigger_id: str) -> CronWorkflows:
        return self._run_coroutine(self.aio_get_cron(cron_trigger_id))

    def schedule_create(
        self,
        workflow_name: str,
        trigger_at: datetime.datetime,
        input: JSONSerializableDict,
        additional_metadata: JSONSerializableDict,
    ) -> ScheduledWorkflows:
        return self._run_coroutine(
            self.aio_create_schedule(
                workflow_name, trigger_at, input, additional_metadata
            )
        )

    def schedule_delete(self, scheduled_trigger_id: str) -> None:
        self._run_coroutine(self.aio_delete_schedule(scheduled_trigger_id))

    def schedule_list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        workflow_id: str | None = None,
        additional_metadata: list[str] | None = None,
        order_by_field: CronWorkflowsOrderByField | None = None,
        order_by_direction: WorkflowRunOrderByDirection | None = None,
    ) -> ScheduledWorkflowsList:
        return self._run_coroutine(
            self.aio_list_schedule(
                offset,
                limit,
                workflow_id,
                additional_metadata,
                order_by_field,
                order_by_direction,
            )
        )

    def schedule_get(self, scheduled_trigger_id: str) -> ScheduledWorkflows:
        return self._run_coroutine(self.aio_get_schedule(scheduled_trigger_id))

    def list_logs(
        self,
        step_run_id: str,
        offset: int | None = None,
        limit: int | None = None,
        levels: list[LogLineLevel] | None = None,
        search: str | None = None,
        order_by_field: LogLineOrderByField | None = None,
        order_by_direction: LogLineOrderByDirection | None = None,
    ) -> LogLineList:
        return self._run_coroutine(
            self.aio_list_logs(
                step_run_id=step_run_id,
                offset=offset,
                limit=limit,
                levels=levels,
                search=search,
                order_by_field=order_by_field,
                order_by_direction=order_by_direction,
            )
        )

    def events_list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        keys: list[str] | None = None,
        workflows: list[str] | None = None,
        statuses: list[WorkflowRunStatus] | None = None,
        search: str | None = None,
        order_by_field: EventOrderByField | None = None,
        order_by_direction: EventOrderByDirection | None = None,
        additional_metadata: list[str] | None = None,
    ) -> EventList:
        return self._run_coroutine(
            self.aio_list_events(
                offset=offset,
                limit=limit,
                keys=keys,
                workflows=workflows,
                statuses=statuses,
                search=search,
                order_by_field=order_by_field,
                order_by_direction=order_by_direction,
                additional_metadata=additional_metadata,
            )
        )

    def events_replay(self, event_ids: list[str] | EventList) -> EventList:
        return self._run_coroutine(self.aio_replay_events(event_ids))
