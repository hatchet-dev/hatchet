import asyncio

from hatchet_sdk.clients.rest.api.log_api import LogApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_log_line_list import V1LogLineList
from hatchet_sdk.clients.v1.api_client import BaseRestClient, retry


class LogsClient(BaseRestClient):
    """
    The logs client is a client for interacting with Hatchet's logs API.
    """

    def _la(self, client: ApiClient) -> LogApi:
        return LogApi(client)

    @retry
    def list(self, task_run_id: str) -> V1LogLineList:
        """
        List log lines for a given task run.

        :param task_run_id: The ID of the task run to list logs for.
        :return: A list of log lines for the specified task run.
        """
        with self.client() as client:
            return self._la(client).v1_log_line_list(task=task_run_id)

    async def aio_list(self, task_run_id: str) -> V1LogLineList:
        """
        List log lines for a given task run.

        :param task_run_id: The ID of the task run to list logs for.
        :return: A list of log lines for the specified task run.
        """
        return await asyncio.to_thread(self.list, task_run_id)
