from __future__ import annotations

import time
from typing import TYPE_CHECKING, Any

from hatchet_sdk.clients.listeners.run_event_listener import (
    RunEventListener,
    RunEventListenerClient,
)
from hatchet_sdk.clients.listeners.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.exceptions import (
    CancellationReason,
    CancelledError,
    FailedTaskRunExceptionGroup,
    TaskRunError,
)
from hatchet_sdk.utils.cancellation import await_with_cancellation

if TYPE_CHECKING:
    from hatchet_sdk.cancellation import CancellationToken
    from hatchet_sdk.clients.admin import AdminClient


class WorkflowRunRef:
    def __init__(
        self,
        workflow_run_id: str,
        workflow_run_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
        admin_client: AdminClient,
    ):
        self.workflow_run_id = workflow_run_id
        self.workflow_run_listener = workflow_run_listener
        self.workflow_run_event_listener = workflow_run_event_listener
        self.admin_client = admin_client

    def __str__(self) -> str:
        return self.workflow_run_id

    def stream(self) -> RunEventListener:
        return self.workflow_run_event_listener.stream(self.workflow_run_id)

    async def aio_result(
        self, cancellation_token: CancellationToken | None = None
    ) -> dict[str, Any]:
        """
        Asynchronously wait for the workflow run to complete and return the result.

        :param cancellation_token: Optional cancellation token to abort the wait.
        :return: A dictionary mapping task names to their outputs.
        """
        if cancellation_token:
            return await await_with_cancellation(
                self.workflow_run_listener.aio_result(self.workflow_run_id),
                cancellation_token,
            )
        return await self.workflow_run_listener.aio_result(self.workflow_run_id)

    def _safely_get_action_name(self, action_id: str | None) -> str | None:
        if not action_id:
            return None

        try:
            return action_id.split(":", maxsplit=1)[1]
        except IndexError:
            return None

    def result(
        self, cancellation_token: CancellationToken | None = None
    ) -> dict[str, Any]:
        """
        Synchronously wait for the workflow run to complete and return the result.

        This method polls the API for the workflow run status. If a cancellation token
        is provided, the polling will be interrupted when cancellation is triggered.

        :param cancellation_token: Optional cancellation token to abort the wait.
        :return: A dictionary mapping task names to their outputs.
        :raises CancelledError: If the cancellation token is triggered.
        :raises FailedTaskRunExceptionGroup: If the workflow run fails.
        :raises ValueError: If the workflow run is not found.
        """
        from hatchet_sdk.clients.admin import RunStatus

        retries = 0

        while True:
            # Check cancellation at start of each iteration
            if cancellation_token and cancellation_token.is_cancelled:
                raise CancelledError(
                    "Operation cancelled by cancellation token",
                    reason=CancellationReason.PARENT_CANCELLED,
                )

            try:
                details = self.admin_client.get_details(self.workflow_run_id)
            except Exception as e:
                retries += 1

                if retries > 10:
                    raise ValueError(
                        f"Workflow run {self.workflow_run_id} not found"
                    ) from e

                # Use interruptible sleep via token.wait()
                if cancellation_token:
                    if cancellation_token.wait(timeout=1.0):
                        raise CancelledError(
                            "Operation cancelled by cancellation token",
                            reason=CancellationReason.PARENT_CANCELLED,
                        ) from None
                else:
                    time.sleep(1)
                continue

            if (
                details.status in [RunStatus.QUEUED, RunStatus.RUNNING]
                or details.done is False
            ):
                # Use interruptible sleep via token.wait()
                if cancellation_token:
                    if cancellation_token.wait(timeout=1.0):
                        raise CancelledError(
                            "Operation cancelled by cancellation token",
                            reason=CancellationReason.PARENT_CANCELLED,
                        )
                else:
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
