import time
from typing import Any

from hatchet_sdk.clients.listeners.run_event_listener import (
    RunEventListener,
    RunEventListenerClient,
)
from hatchet_sdk.clients.listeners.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.features.runs import RunsClient


class WorkflowRunRef:
    def __init__(
        self,
        workflow_run_id: str,
        workflow_run_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
        runs_client: RunsClient,
    ):
        self.workflow_run_id = workflow_run_id
        self.workflow_run_listener = workflow_run_listener
        self.workflow_run_event_listener = workflow_run_event_listener
        self.runs_client = runs_client

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
        retries = 0

        while True:
            try:
                details = self.runs_client.get(self.workflow_run_id)
            except Exception as e:
                retries += 1

                if retries > 10:
                    raise ValueError(
                        f"Workflow run {self.workflow_run_id} not found"
                    ) from e

                time.sleep(1)
                continue

            match details.run.status:
                case V1TaskStatus.RUNNING:
                    time.sleep(1)
                case V1TaskStatus.FAILED:
                    raise ValueError(
                        f"Workflow run failed: {details.run.error_message}"
                    )
                case V1TaskStatus.COMPLETED:
                    return {
                        name: t.output
                        for t in details.tasks
                        if (name := self._safely_get_action_name(t.action_id))
                    }
                case V1TaskStatus.QUEUED:
                    time.sleep(1)
                case V1TaskStatus.CANCELLED:
                    raise ValueError(
                        f"Workflow run cancelled: {details.run.error_message}"
                    )
                case _:
                    raise ValueError(
                        f"Unknown workflow run status: {details.run.status}"
                    )
