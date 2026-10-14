import asyncio
from datetime import datetime, timezone
from typing import cast

from google.protobuf import timestamp_pb2

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
from hatchet_sdk.types.rate_limit import RateLimitDuration
from hatchet_sdk.utils.api_auth import create_authorization_header
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
            metadata=create_authorization_header(self.client_config.token),
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

    def pause(self, key: str, until: datetime) -> datetime:
        """
        Pause a rate limit key, so queued runs that consume it stay queued until the given time.

        :param key: The rendered rate limit key to pause.
        :param until: The time to pause the key until. Naive datetimes are treated as UTC. An earlier time never shortens an existing pause.

        :return: The time the key is paused until.
        """

        paused_until = timestamp_pb2.Timestamp()
        paused_until.FromDatetime(until)

        conn = new_conn(self.client_config, False)
        client = WorkflowServiceStub(conn)
        pause_rate_limit = tenacity_retry(
            client.PauseRateLimit, self.client_config.tenacity
        )

        response = cast(
            v0_workflow_protos.PauseRateLimitResponse,
            pause_rate_limit(
                v0_workflow_protos.PauseRateLimitRequest(
                    key=key,
                    paused_until=paused_until,
                ),
                metadata=create_authorization_header(self.client_config.token),
            ),
        )

        return response.paused_until.ToDatetime(tzinfo=timezone.utc)

    async def aio_pause(self, key: str, until: datetime) -> datetime:
        """
        Pause a rate limit key, so queued runs that consume it stay queued until the given time.

        :param key: The rendered rate limit key to pause.
        :param until: The time to pause the key until. Naive datetimes are treated as UTC. An earlier time never shortens an existing pause.

        :return: The time the key is paused until.
        """

        return await asyncio.to_thread(self.pause, key, until)

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
