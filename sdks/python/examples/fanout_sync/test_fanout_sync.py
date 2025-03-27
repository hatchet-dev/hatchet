from examples.fanout_sync.worker import ParentInput, sync_fanout_parent
from hatchet_sdk import Hatchet


def test_run(hatchet: Hatchet) -> None:
    N = 2

    result = sync_fanout_parent.run(ParentInput(n=N))

    assert len(result.children) == N
