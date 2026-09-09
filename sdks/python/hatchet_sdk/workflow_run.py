import asyncio
import time
from datetime import timedelta
from typing import TYPE_CHECKING, Any

from hatchet_sdk.clients.listeners.run_event_listener import (
    RunEventListener,
    RunEventListenerClient,
)
from hatchet_sdk.clients.listeners.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.exceptions import FailedTaskRunExceptionGroup, TaskRunError

if TYPE_CHECKING:
    from hatchet_sdk.clients.admin import AdminClient

POLL_INTERVAL_SECONDS = 1
MAX_FETCH_RETRIES = 10


class WorkflowRunRef:
    def __init__(
        self,
        workflow_run_id: str,
        workflow_run_listener: PooledWorkflowRunListener,
        workflow_run_event_listener: RunEventListenerClient,
        admin_client: "AdminClient",
    ) -> None:
        self._workflow_run_id = workflow_run_id
        self._workflow_run_listener = workflow_run_listener
        self._workflow_run_event_listener = workflow_run_event_listener
        self._admin_client = admin_client

    def __str__(self) -> str:
        return self.workflow_run_id

    @property
    def workflow_run_id(self) -> str:
        return self._workflow_run_id

    def stream(self) -> RunEventListener:
        """
        Subscribe to the events emitted by the run, such as stream chunks sent via `ctx.put_stream` and run state changes.

        :return: A `RunEventListener` which can be iterated over asynchronously, yielding a `TaskRunEvent` per event.
        """
        return self._workflow_run_event_listener.stream(self.workflow_run_id)

    async def aio_result(
        self,
        timeout: timedelta | None = None,  # noqa: ASYNC109
    ) -> dict[str, Any]:
        """
        Wait for the workflow run to complete and return its result.

        :param timeout: The maximum time to wait for the run to complete. Waits indefinitely if not provided.
        :return: A dictionary mapping each task name in the run to its output.
        :raises TimeoutError: If the run does not complete within the timeout.
        """
        coro = self._workflow_run_listener.aio_result(self.workflow_run_id)

        if timeout is None:
            return await coro

        try:
            return await asyncio.wait_for(coro, timeout=timeout.total_seconds())
        except TimeoutError:
            raise TimeoutError(
                f"Timed out waiting for workflow run {self.workflow_run_id} to complete after {timeout}."
            ) from None

    def result(self, timeout: timedelta | None = None) -> dict[str, Any]:
        """
        Wait for the workflow run to complete and return its result, polling for completion.

        :param timeout: The maximum time to wait for the run to complete. Waits indefinitely if not provided.
        :return: A dictionary mapping each task name in the run to its output.
        :raises TimeoutError: If the run does not complete within the timeout.
        :raises RuntimeError: If fetching the run's status fails repeatedly.
        :raises FailedTaskRunExceptionGroup: If the run fails.
        :raises ValueError: If the run is cancelled or in an unexpected state.
        """
        from hatchet_sdk.clients.admin import RunStatus

        deadline = (
            time.monotonic() + timeout.total_seconds() if timeout is not None else None
        )
        fetch_failures = 0

        while True:
            if deadline is not None and time.monotonic() > deadline:
                raise TimeoutError(
                    f"Timed out waiting for workflow run {self.workflow_run_id} to complete after {timeout}."
                )

            try:
                details = self._admin_client.get_details(self.workflow_run_id)
            except Exception as e:
                fetch_failures += 1

                if fetch_failures > MAX_FETCH_RETRIES:
                    raise RuntimeError(
                        f"Failed to fetch workflow run {self.workflow_run_id} after {MAX_FETCH_RETRIES} attempts."
                    ) from e

                time.sleep(POLL_INTERVAL_SECONDS)
                continue

            if (
                details.status in [RunStatus.QUEUED, RunStatus.RUNNING]
                or details.done is False
            ):
                time.sleep(POLL_INTERVAL_SECONDS)
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
