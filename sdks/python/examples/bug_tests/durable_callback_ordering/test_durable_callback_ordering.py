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
        assert max(result.mid_invocation_counts) >= 2, (
            "no mid was evicted and replayed; the test did not exercise "
            "callback ordering on replay"
        )
