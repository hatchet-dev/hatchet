import asyncio

from hatchet_sdk.clients.rest.api.rate_limits_api import RateLimitsApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.rate_limit_list import RateLimitList
from hatchet_sdk.clients.rest.models.rate_limit_order_by_direction import (
    RateLimitOrderByDirection,
)
from hatchet_sdk.clients.rest.models.rate_limit_order_by_field import (
    RateLimitOrderByField,
)
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.clients.v1.api_client import BaseRestClient
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts import workflows_pb2 as v0_workflow_protos
from hatchet_sdk.contracts.v1 import workflows_pb2 as workflow_protos
from hatchet_sdk.contracts.workflows_pb2_grpc import WorkflowServiceStub
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.rate_limit import RateLimitDuration
from hatchet_sdk.utils.proto_enums import convert_python_enum_to_proto


class RateLimitsClient(BaseRestClient):
    """
    The rate limits client is a wrapper for Hatchet's gRPC API that makes it easier to work with rate limits in Hatchet.
    """

    def _rla(self, client: ApiClient) -> RateLimitsApi:
        return RateLimitsApi(client)

    def put(
        self,
        key: str,
        limit: int,
        duration: RateLimitDuration = RateLimitDuration.SECOND,
    ) -> None:
        """
        Put a rate limit for a given key.

        :param key: The key to set the rate limit for.
        :param limit: The rate limit to set.
        :param duration: The duration of the rate limit.

        :return: None
        """

        duration_proto = convert_python_enum_to_proto(
            duration, workflow_protos.RateLimitDuration
        )

        conn = new_conn(self.client_config, False)
        client = WorkflowServiceStub(conn)
        put_rate_limit = tenacity_retry(
            client.PutRateLimit, self.client_config.tenacity
        )

        put_rate_limit(
            v0_workflow_protos.PutRateLimitRequest(
                key=key,
                limit=limit,
                duration=duration_proto,  # type: ignore[arg-type]
            ),
            metadata=get_metadata(self.client_config.token),
        )

    async def aio_put(
        self,
        key: str,
        limit: int,
        duration: RateLimitDuration = RateLimitDuration.SECOND,
    ) -> None:
        """
        Put a rate limit for a given key.

        :param key: The key to set the rate limit for.
        :param limit: The rate limit to set.
        :param duration: The duration of the rate limit.

        :return: None
        """

        await asyncio.to_thread(self.put, key, limit, duration)

    def list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        search: str | None = None,
        order_by_field: RateLimitOrderByField | None = None,
        order_by_direction: RateLimitOrderByDirection | None = None,
    ) -> RateLimitList:
        """
        List all rate limits for the tenant.

        :param offset: The number of results to skip.
        :param limit: The maximum number of results to return.
        :param search: A search query to filter rate limits by key.
        :param order_by_field: The field to order results by.
        :param order_by_direction: The direction to order results.
        :return: A list of rate limits.
        """

        with self.client() as client:
            rate_limit_list = tenacity_retry(
                self._rla(client).rate_limit_list,
                self.client_config.tenacity,
            )
            return rate_limit_list(
                tenant=self.client_config.tenant_id,
                offset=offset,
                limit=limit,
                search=search,
                order_by_field=order_by_field,
                order_by_direction=order_by_direction,
            )

    async def aio_list(
        self,
        offset: int | None = None,
        limit: int | None = None,
        search: str | None = None,
        order_by_field: RateLimitOrderByField | None = None,
        order_by_direction: RateLimitOrderByDirection | None = None,
    ) -> RateLimitList:
        """
        List all rate limits for the tenant.

        :param offset: The number of results to skip.
        :param limit: The maximum number of results to return.
        :param search: A search query to filter rate limits by key.
        :param order_by_field: The field to order results by.
        :param order_by_direction: The direction to order results.
        :return: A list of rate limits.
        """

        return await asyncio.to_thread(
            self.list,
            offset=offset,
            limit=limit,
            search=search,
            order_by_field=order_by_field,
            order_by_direction=order_by_direction,
        )
