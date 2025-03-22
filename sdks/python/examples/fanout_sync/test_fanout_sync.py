import pytest

from examples.fanout_sync.worker import ParentInput, parent
from hatchet_sdk import Hatchet, Worker


@pytest.mark.parametrize("worker", ["fanout_sync"], indirect=True)
def test_run(hatchet: Hatchet, worker: Worker) -> None:
    N = 2

    result = parent.run(ParentInput(n=N))

    assert len(result["spawn"]["results"]) == N
