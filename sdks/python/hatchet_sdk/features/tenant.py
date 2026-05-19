import asyncio

from hatchet_sdk.clients.rest.api.tenant_api import TenantApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.tenant import Tenant
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import BaseRestClient


class TenantClient(BaseRestClient):
    """
    The tenant client is a client for interacting with your Tenant.
    """

    def _ta(self, client: ApiClient) -> TenantApi:
        return TenantApi(client)

    def get(self) -> Tenant:
        """
        Get the current tenant.

        :return: The tenant.
        """
        with self.client() as client:
            tenant_get = tenacity_retry(
                self._ta(client).tenant_get, self.client_config.tenacity
            )
            return tenant_get(self.client_config.tenant_id)

    async def aio_get(self) -> Tenant:
        """
        Get the current tenant.

        :return: The tenant.
        """
        return await asyncio.to_thread(self.get)
