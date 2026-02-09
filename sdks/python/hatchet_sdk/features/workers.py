import asyncio

from hatchet_sdk.clients.rest.api.worker_api import WorkerApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.update_worker_request import UpdateWorkerRequest
from hatchet_sdk.clients.rest.models.worker import Worker
from hatchet_sdk.clients.rest.models.worker_list import WorkerList
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import BaseRestClient


class WorkersClient(BaseRestClient):
    """
    The workers client is a client for managing workers programmatically within Hatchet.
    """

    def _wa(self, client: ApiClient) -> WorkerApi:
        return WorkerApi(client)

    def get(self, worker_id: str) -> Worker:
        """
        Get a worker by its ID.

        :param worker_id: The ID of the worker to retrieve.
        :return: The worker.
        """
        with self.client() as client:
            worker_get = tenacity_retry(
                self._wa(client).worker_get, self.client_config.tenacity
            )
            return worker_get(worker_id)

    async def aio_get(self, worker_id: str) -> Worker:
        """
        Get a worker by its ID.

        :param worker_id: The ID of the worker to retrieve.
        :return: The worker.
        """
        return await asyncio.to_thread(self.get, worker_id)

    def list(
        self,
    ) -> WorkerList:
        """
        List all workers in the tenant determined by the client config.

        :return: A list of workers.
        """
        with self.client() as client:
            worker_list = tenacity_retry(
                self._wa(client).worker_list, self.client_config.tenacity
            )
            return worker_list(
                tenant=self.client_config.tenant_id,
            )

    async def aio_list(
        self,
    ) -> WorkerList:
        """
        List all workers in the tenant determined by the client config.

        :return: A list of workers.
        """
        return await asyncio.to_thread(self.list)

    def update(self, worker_id: str, opts: UpdateWorkerRequest) -> Worker:
        """
        Update a worker by its ID.

        :param worker_id: The ID of the worker to update.
        :param opts: The update options.
        :return: The updated worker.
        """
        with self.client() as client:
            return self._wa(client).worker_update(
                worker=worker_id,
                update_worker_request=opts,
            )

    async def aio_update(self, worker_id: str, opts: UpdateWorkerRequest) -> Worker:
        """
        Update a worker by its ID.

        :param worker_id: The ID of the worker to update.
        :param opts: The update options.
        :return: The updated worker.
        """
        return await asyncio.to_thread(self.update, worker_id, opts)
