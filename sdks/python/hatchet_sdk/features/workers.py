import asyncio

from hatchet_sdk.clients.rest.api.worker_api import WorkerApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.update_worker_request import UpdateWorkerRequest
from hatchet_sdk.clients.rest.models.worker import Worker
from hatchet_sdk.clients.rest.models.worker_list import WorkerList
from hatchet_sdk.clients.v1.api_client import BaseRestClient


class WorkersClient(BaseRestClient):
    def _wa(self, client: ApiClient) -> WorkerApi:
        return WorkerApi(client)

    def get(self, worker_id: str) -> Worker:
        with self.client() as client:
            return self._wa(client).worker_get(worker_id)

    async def aio_get(self, worker_id: str) -> Worker:
        return await asyncio.to_thread(self.get, worker_id)

    def list(
        self,
    ) -> WorkerList:
        with self.client() as client:
            return self._wa(client).worker_list(
                tenant=self.client_config.tenant_id,
            )

    async def aio_list(
        self,
    ) -> WorkerList:
        return await asyncio.to_thread(self.list)

    def update(self, worker_id: str, opts: UpdateWorkerRequest) -> Worker:
        with self.client() as client:
            return self._wa(client).worker_update(
                worker=worker_id,
                update_worker_request=opts,
            )

    async def aio_update(self, worker_id: str, opts: UpdateWorkerRequest) -> Worker:
        return await asyncio.to_thread(self.update, worker_id, opts)
