import asyncio
from typing import Generic, TypeVar

from pydantic import BaseModel

from hatchet_sdk.clients.run_event_listener import (
    RunEventListener,
    RunEventListenerClient,
)
from hatchet_sdk.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.utils.aio_utils import get_active_event_loop

TWorkflowOutput = TypeVar("TWorkflowOutput", bound=BaseModel)


class WorkflowRunRef(Generic[TWorkflowOutput]):
    def __init__(
        self,
        workflow_run_id: str,
        workflow_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
        output_validator: "TWorkflowOutput",
    ):
        self.workflow_run_id = workflow_run_id
        self.workflow_listener = workflow_listener
        self.workflow_run_event_listener = workflow_run_event_listener
        self.output_validator = output_validator

    def __str__(self) -> str:
        return self.workflow_run_id

    def stream(self) -> RunEventListener:
        return self.workflow_run_event_listener.stream(self.workflow_run_id)

    async def aio_result(self) -> "TWorkflowOutput":
        result = await self.workflow_listener.result(self.workflow_run_id)

        print("\n\nResult", result)

        return self.output_validator.model_validate(result)

    def result(self) -> "TWorkflowOutput":
        coro = self.workflow_listener.result(self.workflow_run_id)

        loop = get_active_event_loop()

        if loop is None:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            try:
                result = loop.run_until_complete(coro)

                return self.output_validator.model_validate(result)
            finally:
                asyncio.set_event_loop(None)
        else:
            result = loop.run_until_complete(coro)

            return self.output_validator.model_validate(result)
