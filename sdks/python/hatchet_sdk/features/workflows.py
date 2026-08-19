import asyncio
from datetime import timedelta
from typing import Literal

from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi
from hatchet_sdk.clients.rest.api.workflow_run_api import WorkflowRunApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.pause_workflow_request import PauseWorkflowRequest
from hatchet_sdk.clients.rest.models.pause_workflow_request_pause import (
    PauseWorkflowRequestPause,
)
from hatchet_sdk.clients.rest.models.pause_workflow_request_unpause import (
    PauseWorkflowRequestUnpause,
)
from hatchet_sdk.clients.rest.models.workflow import Workflow
from hatchet_sdk.clients.rest.models.workflow_list import WorkflowList
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

    def _wra(self, client: ApiClient) -> WorkflowRunApi:
        return WorkflowRunApi(client)

    def _wa(self, client: ApiClient) -> WorkflowApi:
        return WorkflowApi(client)

    async def aio_get(self, workflow_id: str) -> Workflow:
        """
        Get a workflow by its ID.

        :param workflow_id: The ID of the workflow to retrieve.
        :return: The workflow.
        """
        return await asyncio.to_thread(self.get, workflow_id)

    def get(self, workflow_id: str) -> Workflow:
        """
        Get a workflow by its ID.

        :param workflow_id: The ID of the workflow to retrieve.
        :return: The workflow.
        """
        with self.client() as client:
            workflow_get = tenacity_retry(
                self._wa(client).workflow_get, self.client_config.tenacity
            )
            return workflow_get(workflow_id)

    def list(
        self,
        workflow_name: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> WorkflowList:
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
            return workflow_list(
                tenant=self.client_config.tenant_id,
                limit=limit,
                offset=offset,
                name=self.client_config.apply_namespace(workflow_name),
            )

    async def aio_list(
        self,
        workflow_name: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> WorkflowList:
        """
        List all workflows in the tenant determined by the client config that match optional filters.

        :param workflow_name: The name of the workflow to filter by.
        :param limit: The maximum number of items to return.
        :param offset: The offset to start the list from.

        :return: A list of workflows.
        """
        return await asyncio.to_thread(self.list, workflow_name, limit, offset)

    def get_version(
        self, workflow_id: str, version: str | None = None
    ) -> WorkflowVersion:
        """
        Get a workflow version by the workflow ID and an optional version.

        :param workflow_id: The ID of the workflow to retrieve the version for.
        :param version: The version of the workflow to retrieve. If None, the latest version is returned.
        :return: The workflow version.
        """
        with self.client() as client:
            workflow_get_version = tenacity_retry(
                self._wa(client).workflow_version_get, self.client_config.tenacity
            )
            return workflow_get_version(workflow_id, version)

    async def aio_get_version(
        self, workflow_id: str, version: str | None = None
    ) -> WorkflowVersion:
        """
        Get a workflow version by the workflow ID and an optional version.

        :param workflow_id: The ID of the workflow to retrieve the version for.
        :param version: The version of the workflow to retrieve. If None, the latest version is returned.
        :return: The workflow version.
        """
        return await asyncio.to_thread(self.get_version, workflow_id, version)

    def delete(self, workflow_id: str) -> None:
        """
        Permanently delete a workflow.

        **DANGEROUS: This will delete a workflow and all of its data**

        :param workflow_id: The ID of the workflow to delete.
        :return: None
        """

        with self.client() as client:
            return self._wa(client).workflow_delete(workflow_id)

    async def aio_delete(self, workflow_id: str) -> None:
        """
        Permanently delete a workflow.

        **DANGEROUS: This will delete a workflow and all of its data**

        :param workflow_id: The ID of the workflow to delete.
        :return: None
        """

        return await asyncio.to_thread(self.delete, workflow_id)

    def pause(
        self,
        workflow_id: str,
        queue_ttl: timedelta,
        paused_workflow_cron_run_queue_behavior: Literal["DROP", "QUEUE"] = "QUEUE",
        paused_workflow_scheduled_run_queue_behavior: Literal[
            "DROP", "QUEUE"
        ] = "QUEUE",
    ) -> Workflow:
        """
        Pause a workflow.

        :param workflow_id: The ID of the workflow to pause.
        :param queue_ttl: The TTL for queued runs while the workflow is paused before they get dropped.
        :param paused_workflow_cron_run_queue_behavior: The behavior of cron runs triggered while the workflow is paused.
        :param paused_workflow_scheduled_run_queue_behavior: The behavior of scheduled runs triggered while the workflow is paused.
        :return: The updated workflow.
        """
        ttl_expr = timedelta_to_expr(queue_ttl)

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
        workflow_id: str,
        queue_ttl: timedelta,
        paused_workflow_cron_run_queue_behavior: Literal["DROP", "QUEUE"] = "QUEUE",
        paused_workflow_scheduled_run_queue_behavior: Literal[
            "DROP", "QUEUE"
        ] = "QUEUE",
    ) -> Workflow:
        """
        Pause a workflow.

        :param workflow_id: The ID of the workflow to pause.
        :param queue_ttl: The TTL for queued runs while the workflow is paused before they get dropped.
        :param paused_workflow_cron_run_queue_behavior: The behavior of cron runs triggered while the workflow is paused.
        :param paused_workflow_scheduled_run_queue_behavior: The behavior of scheduled runs triggered while the workflow is paused.
        :return: The updated workflow.
        """

        return await asyncio.to_thread(
            self.pause,
            workflow_id,
            queue_ttl,
            paused_workflow_cron_run_queue_behavior,
            paused_workflow_scheduled_run_queue_behavior,
        )

    def unpause(self, workflow_id: str) -> Workflow:
        """
        Unpause a workflow.

        :param workflow_id: The ID of the workflow to unpause.
        :return: The updated workflow.
        """
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

    async def aio_unpause(self, workflow_id: str) -> Workflow:
        """
        Unpause a workflow.

        :param workflow_id: The ID of the workflow to unpause.
        :return: The updated workflow.
        """

        return await asyncio.to_thread(self.unpause, workflow_id)
