import asyncio

import pytest

from examples.bug_tests.durable_callback_ordering.worker import (
    RootInput,
    callback_ordering_root,
)


def _exception_text(error: BaseException) -> str:
    parts = [f"{type(error).__name__}: {error}"]
    if isinstance(error, BaseExceptionGroup):
        parts.extend(_exception_text(nested) for nested in error.exceptions)
    if error.__cause__ is not None:
        parts.append(_exception_text(error.__cause__))
    if error.__context__ is not None and error.__context__ is not error.__cause__:
        parts.append(_exception_text(error.__context__))
    return "\n".join(parts)


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
        evidence = _exception_text(error)
        if "NonDeterminismError" in evidence:
            pytest.fail(
                "replayed completions were consumed out of recorded order:\n" + evidence
            )
        raise

    for result in results:
        assert sorted(result.completed_mids) == list(range(input.durables))
        assert len(result.mid_invocation_counts) == input.durables
        assert max(result.mid_invocation_counts) >= 2, (
            "no mid was evicted and replayed; the test did not exercise "
            "callback ordering on replay"
        )
