import asyncio

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

    @tenacity_retry
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

        client.PutRateLimit(
            v0_workflow_protos.PutRateLimitRequest(
                key=key,
                limit=limit,
                duration=duration_proto,  # type: ignore[arg-type]
            ),
            metadata=get_metadata(self.client_config.token),
        )

    @tenacity_retry
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
