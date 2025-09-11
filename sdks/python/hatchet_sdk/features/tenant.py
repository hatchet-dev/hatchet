import asyncio

from hatchet_sdk.clients.rest.api.tenant_api import TenantApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.tenant import Tenant
from hatchet_sdk.clients.v1.api_client import BaseRestClient, retry


class TenantClient(BaseRestClient):
    """
    The tenant client is a client for interacting with your Tenant.
    """

    def _ta(self, client: ApiClient) -> TenantApi:
        return TenantApi(client)

    @retry
    def get(self) -> Tenant:
        """
        Get the current tenant.

        :return: The tenant.
        """
        with self.client() as client:
            return self._ta(client).tenant_get(self.client_config.tenant_id)

    async def aio_get(self) -> Tenant:
        """
        Get the current tenant.

        :return: The tenant.
        """
        return await asyncio.to_thread(self.get)
