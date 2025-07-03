import asyncio

from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi
from hatchet_sdk.clients.rest.api.workflow_run_api import WorkflowRunApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.workflow import Workflow
from hatchet_sdk.clients.rest.models.workflow_list import WorkflowList
from hatchet_sdk.clients.rest.models.workflow_version import WorkflowVersion
from hatchet_sdk.clients.v1.api_client import BaseRestClient, retry


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

    @retry
    def get(self, workflow_id: str) -> Workflow:
        """
        Get a workflow by its ID.

        :param workflow_id: The ID of the workflow to retrieve.
        :return: The workflow.
        """
        with self.client() as client:
            return self._wa(client).workflow_get(workflow_id)

    @retry
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
            return self._wa(client).workflow_list(
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

    @retry
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
            return self._wa(client).workflow_version_get(workflow_id, version)

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
