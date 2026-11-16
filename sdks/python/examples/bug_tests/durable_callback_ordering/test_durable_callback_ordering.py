import asyncio

import pytest

from examples.bug_tests.durable_callback_ordering.worker import (
    RootInput,
    callback_ordering_root,
)


@pytest.mark.asyncio(loop_scope="session")
async def test_replayed_completions_resume_in_recorded_order() -> None:
    input = RootInput()

    try:
        results = await asyncio.wait_for(
            callback_ordering_root.aio_run_many(
                [callback_ordering_root.create_bulk_run_item(input) for _ in range(25)]
            ),
            timeout=60,
        )
    except Exception as error:
        if "NonDeterminismError" in str(error):
            pytest.fail(
                "replayed completions were consumed out of recorded order:\n"
                + str(error)
            )
        raise

    for result in results:
        assert sorted(result.completed_mids) == list(range(input.durables))
        assert len(result.mid_invocation_counts) == input.durables

    # The worker's TTL sweep evicts one waiting durable run per server round
    # trip, oldest wait first, so under load it does not reach every mid before
    # that mid completes. Which mids get replayed is not guaranteed; that at
    # least one does is, since the sweep starts on the oldest waiting mid within
    # a second and a mid idles for far longer than one round trip.
    replayed_mids = sum(
        1 for result in results for count in result.mid_invocation_counts if count >= 2
    )
    assert replayed_mids > 0, (
        "no mid was evicted and replayed; the test did not exercise "
        "callback ordering on replay"
    )
