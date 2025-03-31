from typing import Any

from hatchet_sdk.clients.run_event_listener import (
    RunEventListener,
    RunEventListenerClient,
)
from hatchet_sdk.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.utils.aio import run_async_from_sync


class WorkflowRunRef:
    def __init__(
        self,
        workflow_run_id: str,
        config: ClientConfig,
    ):
        self.workflow_run_id = workflow_run_id
        self.workflow_listener = PooledWorkflowRunListener(config)
        self.workflow_run_event_listener = RunEventListenerClient(config=config)

    def __str__(self) -> str:
        return self.workflow_run_id

    def stream(self) -> RunEventListener:
        return self.workflow_run_event_listener.stream(self.workflow_run_id)

    async def aio_result(self) -> dict[str, Any]:
        return await self.workflow_listener.aio_result(self.workflow_run_id)

    def result(self) -> dict[str, Any]:
        return run_async_from_sync(self.aio_result)
