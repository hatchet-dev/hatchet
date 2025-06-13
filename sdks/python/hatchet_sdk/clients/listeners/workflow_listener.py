import json
from collections.abc import AsyncIterator
from typing import Any, cast

import grpc
import grpc.aio

from hatchet_sdk.clients.listeners.pooled_listener import PooledListener
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.dispatcher_pb2 import (
    SubscribeToWorkflowRunsRequest,
    WorkflowRunEvent,
)
from hatchet_sdk.contracts.dispatcher_pb2_grpc import DispatcherStub
from hatchet_sdk.exceptions import (
    DedupeViolationError,
    FailedTaskRunExceptionGroup,
    TaskRunError,
)

DEDUPE_MESSAGE = "DUPLICATE_WORKFLOW_RUN"


class PooledWorkflowRunListener(
    PooledListener[SubscribeToWorkflowRunsRequest, WorkflowRunEvent, DispatcherStub]
):
    def create_request_body(self, item: str) -> SubscribeToWorkflowRunsRequest:
        return SubscribeToWorkflowRunsRequest(
            workflowRunId=item,
        )

    def generate_key(self, response: WorkflowRunEvent) -> str:
        return response.workflowRunId

    async def aio_result(self, id: str) -> dict[str, Any]:
        event = await self.subscribe(id)
        errors = [result.error for result in event.results if result.error]
        workflow_run_id = event.workflowRunId

        if errors:
            if DEDUPE_MESSAGE in errors[0]:
                raise DedupeViolationError(errors[0])

            raise FailedTaskRunExceptionGroup(
                f"Workflow run {workflow_run_id} failed.",
                [TaskRunError.deserialize(e) for e in errors],
            )

        return {
            result.stepReadableId: json.loads(result.output)
            for result in event.results
            if result.output
        }

    async def create_subscription(
        self,
        request: AsyncIterator[SubscribeToWorkflowRunsRequest],
        metadata: tuple[tuple[str, str]],
    ) -> grpc.aio.UnaryStreamCall[SubscribeToWorkflowRunsRequest, WorkflowRunEvent]:
        if self.client is None:
            conn = new_conn(self.config, True)
            self.client = DispatcherStub(conn)

        return cast(
            grpc.aio.UnaryStreamCall[SubscribeToWorkflowRunsRequest, WorkflowRunEvent],
            self.client.SubscribeToWorkflowRuns(
                request,  # type: ignore[arg-type]
                metadata=metadata,
            ),
        )
