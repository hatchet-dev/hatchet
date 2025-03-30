from hatchet_sdk.clients.rest.api.log_api import LogApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_log_line_list import V1LogLineList
from hatchet_sdk.clients.v1.api_client import BaseRestClient
from hatchet_sdk.utils.aio import run_async_from_sync


class LogsClient(BaseRestClient):
    def _la(self, client: ApiClient) -> LogApi:
        return LogApi(client)

    async def aio_list(self, task_run_id: str) -> V1LogLineList:
        async with self.client() as client:
            return await self._la(client).v1_log_line_list(task=task_run_id)

    def list(self, task_run_id: str) -> V1LogLineList:
        return run_async_from_sync(self.aio_list, task_run_id)
