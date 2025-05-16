import asyncio

from hatchet_sdk.clients.rest.api.filter_api import FilterApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_create_filter_request import (
    V1CreateFilterRequest,
)
from hatchet_sdk.clients.rest.models.v1_filter import V1Filter
from hatchet_sdk.clients.rest.models.v1_filter_list import V1FilterList
from hatchet_sdk.clients.v1.api_client import BaseRestClient
from hatchet_sdk.utils.typing import JSONSerializableMapping


class FiltersClient(BaseRestClient):
    """
    The filters client is a client for interacting with Hatchet's filters API.
    """

    def _fa(self, client: ApiClient) -> FilterApi:
        return FilterApi(client)

    async def aio_list(
        self,
        limit: int | None = None,
        offset: int | None = None,
        workflow_id_scope_pairs: list[tuple[str, str]] | None = None,
    ) -> V1FilterList:
        """
        List filters for a given tenant.

        :param limit: The maximum number of filters to return.
        :param offset: The number of filters to skip before starting to collect the result set.
        :param workflow_id_scope_pairs: A list of tuples containing workflow IDs and scopes to filter by. The workflow id is first, then the scope is second.

        :return: A list of filters matching the specified criteria.
        """
        return await asyncio.to_thread(
            self.list, limit, offset, workflow_id_scope_pairs
        )

    def list(
        self,
        limit: int | None = None,
        offset: int | None = None,
        workflow_id_scope_pairs: list[tuple[str, str]] | None = None,
    ) -> V1FilterList:
        """
        List filters for a given tenant.

        :param limit: The maximum number of filters to return.
        :param offset: The number of filters to skip before starting to collect the result set.
        :param workflow_id_scope_pairs: A list of tuples containing workflow IDs and scopes to filter by. The workflow id is first, then the scope is second.

        :return: A list of filters matching the specified criteria.
        """
        workflow_ids = (
            [pair[0] for pair in workflow_id_scope_pairs]
            if workflow_id_scope_pairs
            else None
        )
        scopes = (
            [pair[1] for pair in workflow_id_scope_pairs]
            if workflow_id_scope_pairs
            else None
        )

        with self.client() as client:
            return self._fa(client).v1_filter_list(
                tenant=self.tenant_id,
                limit=limit,
                offset=offset,
                workflow_ids=workflow_ids,
                scopes=scopes,
            )

    def get(
        self,
        filter_id: str,
    ) -> V1Filter:
        """
        Get a filter by its ID.

        :param filter_id: The ID of the filter to retrieve.

        :return: The filter with the specified ID.
        """
        with self.client() as client:
            return self._fa(client).v1_filter_get(
                tenant=self.tenant_id,
                v1_filter=filter_id,
            )

    async def aio_get(
        self,
        filter_id: str,
    ) -> V1Filter:
        """
        Get a filter by its ID.

        :param filter_id: The ID of the filter to retrieve.

        :return: The filter with the specified ID.
        """
        return await asyncio.to_thread(self.get, filter_id)

    def create(
        self,
        workflow_id: str,
        expression: str,
        scope: str,
        payload: JSONSerializableMapping = {},
    ) -> V1Filter:
        """
        Create a new filter.

        :param workflow_id: The ID of the workflow to associate with the filter.
        :param expression: The expression to evaluate for the filter.
        :param scope: The scope for the filter.
        :param payload: The payload to send with the filter.

        :return: The created filter.
        """
        with self.client() as client:
            return self._fa(client).v1_filter_create(
                tenant=self.tenant_id,
                v1_create_filter_request=V1CreateFilterRequest(
                    workflowId=workflow_id,
                    expression=expression,
                    scope=scope,
                    payload=dict(payload),
                ),
            )

    async def aio_create(
        self,
        workflow_id: str,
        expression: str,
        scope: str,
        payload: JSONSerializableMapping = {},
    ) -> V1Filter:
        """
        Create a new filter.

        :param workflow_id: The ID of the workflow to associate with the filter.
        :param expression: The expression to evaluate for the filter.
        :param scope: The scope for the filter.
        :param payload: The payload to send with the filter.

        :return: The created filter.
        """
        return await asyncio.to_thread(
            self.create, workflow_id, expression, scope, payload
        )

    def delete(
        self,
        filter_id: str,
    ) -> V1Filter:
        """
        Delete a filter by its ID.

        :param filter_id: The ID of the filter to delete.
        :return: The deleted filter.
        """
        with self.client() as client:
            return self._fa(client).v1_filter_delete(
                tenant=self.tenant_id,
                v1_filter=filter_id,
            )

    async def aio_delete(
        self,
        filter_id: str,
    ) -> V1Filter:
        """
        Delete a filter by its ID.

        :param filter_id: The ID of the filter to delete.
        :return: The deleted filter.
        """
        return await asyncio.to_thread(self.delete, filter_id)
