class NonRetryableException(Exception):  # noqa: N818
    pass


class DedupeViolationError(Exception):
    """Raised by the Hatchet library to indicate that a workflow has already been run with this deduplication value."""


class FailedWorkflowRunError(Exception):
    def __init__(self, workflow_run_id: str, message: str) -> None:
        self.workflow_run_id = workflow_run_id
        self.message = message

        super().__init__(f"Workflow run {workflow_run_id} failed: {message}")

    def __str__(self) -> str:
        return f"Workflow run {self.workflow_run_id} failed: {self.message}"


class LoopAlreadyRunningError(Exception):
    pass
