from examples.fanout_sync.worker import ParentInput, sync_fanout_parent

def test_run() -> None:
    N = 2

    result = sync_fanout_parent.run(ParentInput(n=N))

    assert len(result["spawn"]["results"]) == N
