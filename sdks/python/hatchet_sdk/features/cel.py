import asyncio
from typing import Literal

from pydantic import BaseModel, Field

from hatchet_sdk.clients.rest.api.cel_api import CELApi
from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.models.v1_cel_debug_request import V1CELDebugRequest
from hatchet_sdk.clients.rest.models.v1_cel_debug_response_status import (
    V1CELDebugResponseStatus,
)
from hatchet_sdk.clients.v1.api_client import BaseRestClient, retry
from hatchet_sdk.utils.typing import JSONSerializableMapping


class CELSuccess(BaseModel):
    status: Literal["success"] = "success"
    output: bool


class CELFailure(BaseModel):
    status: Literal["failure"] = "failure"
    error: str


class CELEvaluationResult(BaseModel):
    result: CELSuccess | CELFailure = Field(discriminator="status")


class CELClient(BaseRestClient):
    """
    The CEL client is a client for debugging CEL expressions within Hatchet
    """

    def _ca(self, client: ApiClient) -> CELApi:
        return CELApi(client)

    @retry
    def debug(
        self,
        expression: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping | None = None,
        filter_payload: JSONSerializableMapping | None = None,
    ) -> CELEvaluationResult:
        """
        Debug a CEL expression with the provided input, filter payload, and optional metadata. Useful for testing and validating CEL expressions and debugging issues in production.

        :param expression: The CEL expression to debug.
        :param input: The input, which simulates the workflow run input.
        :param additional_metadata: Additional metadata, which simulates metadata that could be sent with an event or a workflow run
        :param filter_payload: The filter payload, which simulates a payload set on a previous-created filter

        :raises ValueError: If no response is received from the CEL debug API.

        :return: A V1CELDebugErrorResponse or V1CELDebugSuccessResponse containing the result of the debug operation.
        """
        request = V1CELDebugRequest(
            expression=expression,
            input=input,
            additionalMetadata=additional_metadata,
            filterPayload=filter_payload,
        )
        with self.client() as client:
            result = self._ca(client).v1_cel_debug(
                tenant=self.client_config.tenant_id, v1_cel_debug_request=request
            )

            if result.status == V1CELDebugResponseStatus.ERROR:
                if result.error is None:
                    raise ValueError("No error message received from CEL debug API.")

                return CELEvaluationResult(result=CELFailure(error=result.error))

            if result.output is None:
                raise ValueError("No output received from CEL debug API.")

            return CELEvaluationResult(result=CELSuccess(output=result.output))

    async def aio_debug(
        self,
        expression: str,
        input: JSONSerializableMapping,
        additional_metadata: JSONSerializableMapping | None = None,
        filter_payload: JSONSerializableMapping | None = None,
    ) -> CELEvaluationResult:
        """
        Debug a CEL expression with the provided input, filter payload, and optional metadata. Useful for testing and validating CEL expressions and debugging issues in production.

        :param expression: The CEL expression to debug.
        :param input: The input, which simulates the workflow run input.
        :param additional_metadata: Additional metadata, which simulates metadata that could be sent with an event or a workflow run
        :param filter_payload: The filter payload, which simulates a payload set on a previous-created filter

        :return: A V1CELDebugErrorResponse or V1CELDebugSuccessResponse containing the result of the debug operation.
        """
        return await asyncio.to_thread(
            self.debug, expression, input, additional_metadata, filter_payload
        )
