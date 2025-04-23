from typing import TYPE_CHECKING, Any

from hatchet_sdk.clients.listeners.run_event_listener import (
    RunEventListener,
    RunEventListenerClient,
)
from hatchet_sdk.clients.listeners.workflow_listener import PooledWorkflowRunListener

if TYPE_CHECKING:
    from hatchet_sdk.clients.admin import AdminClient


class WorkflowRunRef:
    def __init__(
        self,
        workflow_run_id: str,
        workflow_run_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
        admin_client: "AdminClient",
    ):
        self.workflow_run_id = workflow_run_id
        self.workflow_run_listener = workflow_run_listener
        self.workflow_run_event_listener = workflow_run_event_listener
        self.admin_client = admin_client

    def __str__(self) -> str:
        return self.workflow_run_id

    def stream(self) -> RunEventListener:
        return self.workflow_run_event_listener.stream(self.workflow_run_id)

    async def aio_result(self) -> dict[str, Any]:
        return await self.workflow_run_listener.aio_result(self.workflow_run_id)

    def _safely_get_action_name(self, action_id: str | None) -> str | None:
        if not action_id:
            return None

        try:
            return action_id.split(":", maxsplit=1)[1]
        except IndexError:
            return None

    def result(self) -> dict[str, Any]:
        return self.admin_client.get_workflow_run_output(self.workflow_run_id)
