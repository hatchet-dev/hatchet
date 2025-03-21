import pytest

from examples.fanout_sync.worker import ParentInput, parent
from hatchet_sdk import Hatchet, Worker


@pytest.mark.parametrize("worker", ["fanout_sync"], indirect=True)
def test_run(hatchet: Hatchet, worker: Worker) -> None:
    N = 2

    run = parent.run(ParentInput(n=N))
    result = run.result()

    assert len(result["spawn"]["results"]) == N
