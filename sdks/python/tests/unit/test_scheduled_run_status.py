import pytest

from hatchet_sdk.clients.rest.models.workflow_run_status import WorkflowRunStatus
from hatchet_sdk.features.scheduled import _WORKFLOW_RUN_STATUS_TO_RUN_STATUS


@pytest.mark.parametrize("workflow_run_status", list(WorkflowRunStatus))
def test_every_workflow_run_status_maps_to_a_run_status(
    workflow_run_status: WorkflowRunStatus,
) -> None:
    assert workflow_run_status in _WORKFLOW_RUN_STATUS_TO_RUN_STATUS
