from hatchet_sdk.clients.rest.api.worker_api import WorkerApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.update_worker_request import UpdateWorkerRequest
from hatchet_sdk.clients.rest.models.worker import Worker
from hatchet_sdk.clients.rest.models.worker_list import WorkerList
from hatchet_sdk.clients.v1.api_client import BaseRestClient
from hatchet_sdk.utils.aio import run_async_from_sync


class WorkersClient(BaseRestClient):
    def _wa(self, client: ApiClient) -> WorkerApi:
        return WorkerApi(client)

    async def aio_get(self, worker_id: str) -> Worker:
        async with self.client() as client:
            return await self._wa(client).worker_get(worker_id)

    def get(self, worker_id: str) -> Worker:
        return run_async_from_sync(self.aio_get, worker_id)

    async def aio_list(
        self,
    ) -> WorkerList:
        async with self.client() as client:
            return await self._wa(client).worker_list(
                tenant=self.client_config.tenant_id,
            )

    def list(
        self,
    ) -> WorkerList:
        return run_async_from_sync(self.aio_list)

    async def aio_update(self, worker_id: str, opts: UpdateWorkerRequest) -> Worker:
        async with self.client() as client:
            return await self._wa(client).worker_update(
                worker=worker_id,
                update_worker_request=opts,
            )

    def update(self, worker_id: str, opts: UpdateWorkerRequest) -> Worker:
        return run_async_from_sync(self.aio_update, worker_id, opts)
