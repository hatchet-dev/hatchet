import time
from typing import TYPE_CHECKING, Any

from hatchet_sdk.clients.listeners.run_event_listener import (
    RunEventListener,
    RunEventListenerClient,
)
from hatchet_sdk.clients.listeners.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.exceptions import FailedTaskRunExceptionGroup, TaskRunError
from hatchet_sdk.logger import logger

if TYPE_CHECKING:
    from hatchet_sdk.clients.admin import AdminClient

# Constants for result polling with exponential backoff
_INITIAL_BACKOFF_SECONDS = 0.1
_MAX_BACKOFF_SECONDS = 2.0
_BACKOFF_MULTIPLIER = 2.0
_MAX_RETRIES = 30


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
        """Wait for the workflow run to complete and return the result.

        Uses exponential backoff when polling for workflow run status to handle
        race conditions where the workflow run may not be immediately visible
        after creation (e.g., when spawning child workflows synchronously).

        :returns: A dictionary mapping task readable IDs to their outputs.

        :raises FailedTaskRunExceptionGroup: If the workflow run failed.
        :raises ValueError: If the workflow run was cancelled or could not be found.
        """
        from hatchet_sdk.clients.admin import RunStatus

        retries = 0
        backoff = _INITIAL_BACKOFF_SECONDS

        while True:
            try:
                details = self.admin_client.get_details(self.workflow_run_id)
            except Exception as e:
                retries += 1

                if retries > _MAX_RETRIES:
                    logger.warning(
                        "failed to get workflow run details after %d retries: %s",
                        retries,
                        self.workflow_run_id,
                    )
                    raise ValueError(
                        f"Workflow run {self.workflow_run_id} not found after "
                        f"{retries} retries. This may indicate the workflow run "
                        f"has not propagated yet or does not exist. "
                        f"Last error: {e}"
                    ) from e

                time.sleep(backoff)
                backoff = min(backoff * _BACKOFF_MULTIPLIER, _MAX_BACKOFF_SECONDS)
                continue

            if (
                details.status in [RunStatus.QUEUED, RunStatus.RUNNING]
                or details.done is False
            ):
                time.sleep(1)
                continue

            if details.status == RunStatus.FAILED:
                raise FailedTaskRunExceptionGroup(
                    f"Workflow run {self.workflow_run_id} failed.",
                    [
                        TaskRunError.deserialize(run.error)
                        for run in details.task_runs.values()
                        if run.error
                    ],
                )

            if details.status == RunStatus.COMPLETED:
                return {
                    readable_id: run.output
                    for readable_id, run in details.task_runs.items()
                } or {}

            if details.status == RunStatus.CANCELLED:
                raise ValueError(f"Workflow run {self.workflow_run_id} was cancelled.")

            raise ValueError(
                f"Workflow run {self.workflow_run_id} has not completed yet."
            )
