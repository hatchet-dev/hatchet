import json
from collections.abc import AsyncIterator
from typing import TYPE_CHECKING, Any

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
    CancelledRunError,
    DedupeViolationError,
    FailedTaskRunExceptionGroup,
    TaskRunError,
)

if TYPE_CHECKING:
    from hatchet_sdk.contracts.dispatcher_pb2_grpc import DispatcherAsyncStub

DEDUPE_MESSAGE = "DUPLICATE_WORKFLOW_RUN"


class PooledWorkflowRunListener(
    PooledListener[SubscribeToWorkflowRunsRequest, WorkflowRunEvent]
):
    ## IMPORTANT: This needs to be created lazily so we don't require
    ## an event loop to instantiate the client.
    client: "DispatcherAsyncStub | None" = None

    def create_request_body(self, item: str) -> SubscribeToWorkflowRunsRequest:
        return SubscribeToWorkflowRunsRequest(
            workflow_run_id=item,
        )

    def generate_key(self, response: WorkflowRunEvent) -> str:
        return response.workflow_run_id

    async def aio_result(self, id: str) -> dict[str, Any]:
        event = await self.subscribe(id)
        errors = [result.error for result in event.results if result.error]
        workflow_run_id = event.workflow_run_id

        if errors:
            if DEDUPE_MESSAGE in errors[0]:
                raise DedupeViolationError(errors[0])

            raise FailedTaskRunExceptionGroup(
                f"Workflow run {workflow_run_id} failed.",
                [TaskRunError._deserialize(e) for e in errors],
            )

        ## the engine reports a cancelled task with an error that is set but empty
        if any(result.HasField("error") for result in event.results):
            raise CancelledRunError(workflow_run_id)

        return {
            result.task_name: json.loads(result.output)
            for result in event.results
            if result.output
        }

    async def create_subscription(
        self,
        request: AsyncIterator[SubscribeToWorkflowRunsRequest],
        metadata: tuple[tuple[str, str]],
    ) -> grpc.aio.StreamStreamCall[SubscribeToWorkflowRunsRequest, WorkflowRunEvent]:
        if self.client is None:
            conn = new_conn(self.config, True)
            self.client = DispatcherStub(conn)

        return self.client.SubscribeToWorkflowRuns(
            request,
            metadata=metadata,
        )
