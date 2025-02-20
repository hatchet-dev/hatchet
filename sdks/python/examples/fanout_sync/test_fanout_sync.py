import pytest

from hatchet_sdk import Hatchet, Worker


@pytest.mark.parametrize("worker", ["fanout_sync"], indirect=True)
def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = hatchet.admin.run_workflow("SyncFanoutParent", {"n": 2})
    result = run.sync_result()
    assert len(result["spawn"]["results"]) == 2
