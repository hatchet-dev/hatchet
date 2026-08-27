import asyncio

import pytest

from examples.bug_tests.durable_callback_ordering.worker import (
    RootInput,
    callback_ordering_root,
)


@pytest.mark.asyncio(loop_scope="session")
async def test_replayed_completions_resume_in_recorded_order() -> None:
    """Concurrent durable mids replay after eviction without NonDeterminismError.

    Each mid gathers staggered first-generation children, so second-generation
    spawns are emitted in completion order rather than spawn order. The short
    eviction TTL forces replays mid-flight; the run only completes cleanly if
    cached completions are consumed in the recorded satisfied order.
    """
    params = RootInput()

    result = await asyncio.wait_for(callback_ordering_root.aio_run(params), timeout=300)

    assert sorted(result.completed_mids) == list(range(params.durables))
    assert len(result.mid_invocation_counts) == params.durables
    assert max(result.mid_invocation_counts) >= 2, (
        "no mid was evicted and replayed; the test did not exercise "
        "callback ordering on replay"
    )
