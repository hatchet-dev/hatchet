from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_workflow_run_details import V1WorkflowRunDetails
from hatchet_sdk.clients.v1.api_client import BaseRestClient
from hatchet_sdk.config import ClientConfig


class RunsClient(BaseRestClient):
    def __init__(self, config: ClientConfig) -> None:
        super().__init__(config)

    def _wra(self, client: ApiClient) -> WorkflowRunsApi:
        return WorkflowRunsApi(client)

    async def aio_get(self, workflow_run_id: str) -> V1WorkflowRunDetails:
        async with self.client() as client:
            return await self._wra(client).v1_workflow_run_get(workflow_run_id)

    def get(self, workflow_run_id: str) -> V1WorkflowRunDetails:
        return self._run_async_from_sync(self.aio_get, workflow_run_id)
