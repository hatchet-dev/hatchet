import asyncio
from datetime import datetime

from hatchet_sdk.clients.rest.api.log_api import LogApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_log_line_list import V1LogLineList
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import BaseRestClient


class LogsClient(BaseRestClient):
    """
    The logs client is a client for interacting with Hatchet's logs API.
    """

    def _la(self, client: ApiClient) -> LogApi:
        return LogApi(client)

    def list(
        self,
        task_run_id: str,
        limit: int = 1000,
        since: datetime | None = None,
        until: datetime | None = None,
    ) -> V1LogLineList:
        """
        List log lines for a given task run.

        :param task_run_id: The ID of the task run to list logs for.
        :param limit: Maximum number of log lines to return (default: 1000).
        :param since: The start time to get logs for.
        :param until: The end time to get logs for.
        :return: A list of log lines for the specified task run.
        """
        with self.client() as client:
            v1_log_line_list = tenacity_retry(
                self._la(client).v1_log_line_list, self.client_config.tenacity
            )
            return v1_log_line_list(
                task=task_run_id, limit=limit, since=since, until=until
            )

    async def aio_list(
        self,
        task_run_id: str,
        limit: int = 1000,
        since: datetime | None = None,
        until: datetime | None = None,
    ) -> V1LogLineList:
        """
        List log lines for a given task run.

        :param task_run_id: The ID of the task run to list logs for.
        :param limit: Maximum number of log lines to return (default: 1000).
        :param since: The start time to get logs for.
        :param until: The end time to get logs for.
        :return: A list of log lines for the specified task run.
        """
        return await asyncio.to_thread(self.list, task_run_id, limit, since, until)
