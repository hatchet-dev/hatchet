import asyncio
from datetime import timedelta
from typing import TYPE_CHECKING, Any, Literal

if TYPE_CHECKING:
    from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi
    from hatchet_sdk.clients.rest.api.workflow_run_api import WorkflowRunApi
    from hatchet_sdk.runnables.workflow import BaseWorkflow

from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.exceptions import NotFoundException
from hatchet_sdk.clients.rest.models.pause_workflow_request import PauseWorkflowRequest
from hatchet_sdk.clients.rest.models.pause_workflow_request_pause import (
    PauseWorkflowRequestPause,
)
from hatchet_sdk.clients.rest.models.pause_workflow_request_unpause import (
    PauseWorkflowRequestUnpause,
)
from hatchet_sdk.clients.rest.models.workflow import Workflow
from hatchet_sdk.clients.rest.models.workflow_pause_scheduled_cron_run_queue_behavior import (
    WorkflowPauseScheduledCronRunQueueBehavior,
)
from hatchet_sdk.clients.rest.models.workflow_update_request import (
    WorkflowUpdateRequest,
)
from hatchet_sdk.clients.rest.models.workflow_version import WorkflowVersion
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import BaseRestClient
from hatchet_sdk.utils.timedelta_to_expression import (
    timedelta_to_expr,
)


class WorkflowsClient(BaseRestClient):
    """
    The workflows client is a client for managing workflows programmatically within Hatchet.

    Note that workflows are the declaration, _not_ the individual runs. If you're looking for runs, use the `RunsClient` instead.
    """

    def _wra(self, client: ApiClient) -> "WorkflowRunApi":
        from hatchet_sdk.clients.rest.api.workflow_run_api import WorkflowRunApi

        return WorkflowRunApi(client)

    def _wa(self, client: ApiClient) -> "WorkflowApi":
        from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi

        return WorkflowApi(client)

    def _get_workflow_by_name(self, workflow: "str | BaseWorkflow[Any]") -> Workflow:
        workflow_name = self.client_config.apply_namespace(
            workflow if isinstance(workflow, str) else workflow.name
        )

        matching_workflow = next(
            (
                candidate
                for candidate in self.list(workflow_name=workflow_name)
                if candidate.name == workflow_name
            ),
            None,
        )

        if matching_workflow is None:
            raise NotFoundException(
                status=404, reason=f"No workflow named {workflow_name} was found"
            )

        return matching_workflow

    def _get_workflow_id(self, workflow: "str | BaseWorkflow[Any]") -> str:
        return self._get_workflow_by_name(workflow).metadata.id

    async def aio_get(self, workflow: "str | BaseWorkflow[Any]") -> Workflow:
        """
        Get a workflow by its name.

        :param workflow: The name of the workflow to retrieve, or the workflow itself. Raises `NotFoundException` if no workflow has that name.
        :return: The workflow.
        """
        return await asyncio.to_thread(self.get, workflow)

    def get(self, workflow: "str | BaseWorkflow[Any]") -> Workflow:
        """
        Get a workflow by its name.

        :param workflow: The name of the workflow to retrieve, or the workflow itself. Raises `NotFoundException` if no workflow has that name.
        :return: The workflow.
        """
        workflow_id = self._get_workflow_id(workflow)

        with self.client() as client:
            workflow_get = tenacity_retry(
                self._wa(client).workflow_get, self.client_config.tenacity
            )
            return workflow_get(workflow_id)

    async def aio_list(
        self,
        *,
        workflow_name: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> "list[Workflow]":
        """
        List all workflows in the tenant determined by the client config that match optional filters.

        :param workflow_name: The name of the workflow to filter by.
        :param limit: The maximum number of items to return.
        :param offset: The offset to start the list from.

        :return: A list of workflows.
        """
        return await asyncio.to_thread(
            self.list, workflow_name=workflow_name, limit=limit, offset=offset
        )

    def list(
        self,
        *,
        workflow_name: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> "list[Workflow]":
        """
        List all workflows in the tenant determined by the client config that match optional filters.

        :param workflow_name: The name of the workflow to filter by.
        :param limit: The maximum number of items to return.
        :param offset: The offset to start the list from.

        :return: A list of workflows.
        """
        with self.client() as client:
            workflow_list = tenacity_retry(
                self._wa(client).workflow_list, self.client_config.tenacity
            )
            wl = workflow_list(
                tenant=self.client_config.tenant_id,
                limit=limit,
                offset=offset,
                name=self.client_config.apply_namespace(workflow_name),
            )

            return wl.rows or []

    def get_version(
        self, workflow: "str | BaseWorkflow[Any]", *, version: str | None = None
    ) -> WorkflowVersion:
        """
        Get a version of a workflow.

        :param workflow: The name of the workflow to retrieve the version for, or the workflow itself. Raises `NotFoundException` if no workflow has that name.
        :param version: The version of the workflow to retrieve. If None, the latest version is returned.
        :return: The workflow version.
        """
        workflow_id = self._get_workflow_id(workflow)

        with self.client() as client:
            workflow_get_version = tenacity_retry(
                self._wa(client).workflow_version_get, self.client_config.tenacity
            )
            return workflow_get_version(workflow_id, version)

    async def aio_get_version(
        self, workflow: "str | BaseWorkflow[Any]", *, version: str | None = None
    ) -> WorkflowVersion:
        """
        Get a version of a workflow.

        :param workflow: The name of the workflow to retrieve the version for, or the workflow itself. Raises `NotFoundException` if no workflow has that name.
        :param version: The version of the workflow to retrieve. If None, the latest version is returned.
        :return: The workflow version.
        """
        return await asyncio.to_thread(self.get_version, workflow, version=version)

    def delete(self, workflow: "str | BaseWorkflow[Any]") -> None:
        """
        Permanently delete a workflow.

        **DANGEROUS: This will delete a workflow and all of its data**

        :param workflow: The name of the workflow to delete, or the workflow itself. Raises `NotFoundException` if no workflow has that name.
        :return: None
        """
        workflow_id = self._get_workflow_id(workflow)

        with self.client() as client:
            return self._wa(client).workflow_delete(workflow_id)

    async def aio_delete(self, workflow: "str | BaseWorkflow[Any]") -> None:
        """
        Permanently delete a workflow.

        **DANGEROUS: This will delete a workflow and all of its data**

        :param workflow: The name of the workflow to delete, or the workflow itself. Raises `NotFoundException` if no workflow has that name.
        :return: None
        """

        return await asyncio.to_thread(self.delete, workflow)

    def pause(
        self,
        workflow: "str | BaseWorkflow[Any]",
        queue_ttl: timedelta,
        *,
        paused_workflow_cron_run_queue_behavior: Literal["DROP", "QUEUE"] = "QUEUE",
        paused_workflow_scheduled_run_queue_behavior: Literal[
            "DROP", "QUEUE"
        ] = "QUEUE",
    ) -> Workflow:
        """
        Pause a workflow.

        :param workflow: The name of the workflow to pause, or the workflow itself. Raises `NotFoundException` if no workflow has that name.
        :param queue_ttl: The TTL for queued runs while the workflow is paused before they get dropped.
        :param paused_workflow_cron_run_queue_behavior: The behavior of cron runs triggered while the workflow is paused.
        :param paused_workflow_scheduled_run_queue_behavior: The behavior of scheduled runs triggered while the workflow is paused.
        :return: The updated workflow.
        """
        ttl_expr = timedelta_to_expr(queue_ttl)
        workflow_id = self._get_workflow_id(workflow)

        with self.client() as client:
            workflow_update_pause = tenacity_retry(
                self._wa(client).workflow_update, self.client_config.tenacity
            )
            return workflow_update_pause(
                workflow_id,
                WorkflowUpdateRequest(
                    pause=PauseWorkflowRequest(
                        PauseWorkflowRequestPause(
                            action="pause",
                            pausedWorkflowCronRunQueueBehavior=WorkflowPauseScheduledCronRunQueueBehavior(
                                paused_workflow_cron_run_queue_behavior
                            ),
                            pausedWorkflowScheduledRunQueueBehavior=WorkflowPauseScheduledCronRunQueueBehavior(
                                paused_workflow_scheduled_run_queue_behavior
                            ),
                            pausedWorkflowQueueTTL=ttl_expr,
                        )
                    )
                ),
            )

    async def aio_pause(
        self,
        workflow: "str | BaseWorkflow[Any]",
        queue_ttl: timedelta,
        *,
        paused_workflow_cron_run_queue_behavior: Literal["DROP", "QUEUE"] = "QUEUE",
        paused_workflow_scheduled_run_queue_behavior: Literal[
            "DROP", "QUEUE"
        ] = "QUEUE",
    ) -> Workflow:
        """
        Pause a workflow.

        :param workflow: The name of the workflow to pause, or the workflow itself. Raises `NotFoundException` if no workflow has that name.
        :param queue_ttl: The TTL for queued runs while the workflow is paused before they get dropped.
        :param paused_workflow_cron_run_queue_behavior: The behavior of cron runs triggered while the workflow is paused.
        :param paused_workflow_scheduled_run_queue_behavior: The behavior of scheduled runs triggered while the workflow is paused.
        :return: The updated workflow.
        """

        return await asyncio.to_thread(
            self.pause,
            workflow,
            queue_ttl,
            paused_workflow_cron_run_queue_behavior=paused_workflow_cron_run_queue_behavior,
            paused_workflow_scheduled_run_queue_behavior=paused_workflow_scheduled_run_queue_behavior,
        )

    def unpause(self, workflow: "str | BaseWorkflow[Any]") -> Workflow:
        """
        Unpause a workflow.

        :param workflow: The name of the workflow to unpause, or the workflow itself. Raises `NotFoundException` if no workflow has that name.
        :return: The updated workflow.
        """
        workflow_id = self._get_workflow_id(workflow)

        with self.client() as client:
            workflow_update_pause = tenacity_retry(
                self._wa(client).workflow_update, self.client_config.tenacity
            )
            return workflow_update_pause(
                workflow_id,
                WorkflowUpdateRequest(
                    pause=PauseWorkflowRequest(
                        PauseWorkflowRequestUnpause(action="unpause")
                    )
                ),
            )

    async def aio_unpause(self, workflow: "str | BaseWorkflow[Any]") -> Workflow:
        """
        Unpause a workflow.

        :param workflow: The name of the workflow to unpause, or the workflow itself. Raises `NotFoundException` if no workflow has that name.
        :return: The updated workflow.
        """

        return await asyncio.to_thread(self.unpause, workflow)
