import asyncio

from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi
from hatchet_sdk.clients.rest.api.workflow_run_api import WorkflowRunApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.workflow import Workflow
from hatchet_sdk.clients.rest.models.workflow_list import WorkflowList
from hatchet_sdk.clients.rest.models.workflow_version import WorkflowVersion
from hatchet_sdk.clients.v1.api_client import BaseRestClient


class WorkflowsClient(BaseRestClient):
    def _wra(self, client: ApiClient) -> WorkflowRunApi:
        return WorkflowRunApi(client)

    def _wa(self, client: ApiClient) -> WorkflowApi:
        return WorkflowApi(client)

    async def aio_get(self, workflow_id: str) -> Workflow:
        return await asyncio.to_thread(self.get, workflow_id)

    def get(self, workflow_id: str) -> Workflow:
        with self.client() as client:
            return self._wa(client).workflow_get(workflow_id)

    def list(
        self,
        workflow_name: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> WorkflowList:
        with self.client() as client:
            return self._wa(client).workflow_list(
                tenant=self.client_config.tenant_id,
                limit=limit,
                offset=offset,
                name=workflow_name,
            )

    async def aio_list(
        self,
        workflow_name: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> WorkflowList:
        return await asyncio.to_thread(self.list, workflow_name, limit, offset)

    def get_version(
        self, workflow_id: str, version: str | None = None
    ) -> WorkflowVersion:
        with self.client() as client:
            return self._wa(client).workflow_version_get(workflow_id, version)

    async def aio_get_version(
        self, workflow_id: str, version: str | None = None
    ) -> WorkflowVersion:
        return await asyncio.to_thread(self.get_version, workflow_id, version)
