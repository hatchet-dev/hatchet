from datetime import timedelta
from typing import Any

import pytest

from hatchet_sdk import ConcurrencyStrategy, Context, Hatchet, TaskDefaults
from hatchet_sdk.runnables.workflow import (
    DEFAULT_EXECUTION_TIMEOUT,
    DEFAULT_RETRIES,
    DEFAULT_SCHEDULE_TIMEOUT,
    compute_task_parameters,
)
from hatchet_sdk.types.concurrency import normalize_concurrency_to_strategies

TASK_DEFAULTS = TaskDefaults(
    schedule_timeout=timedelta(seconds=3),
    execution_timeout=timedelta(minutes=10),
    retries=5,
    backoff_factor=2,
    backoff_max_seconds=30,
)

CONCURRENCY_STRATEGY = ConcurrencyStrategy(
    expression="input.key", max_runs=1, strategy="GROUP_ROUND_ROBIN"
)


def test_explicit_values_equal_to_the_built_in_defaults_are_not_overridden() -> None:
    computed = compute_task_parameters(
        schedule_timeout=DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout=DEFAULT_EXECUTION_TIMEOUT,
        retries=DEFAULT_RETRIES,
        backoff_factor=None,
        backoff_max_seconds=None,
        task_defaults=TASK_DEFAULTS,
    )

    assert computed.schedule_timeout == DEFAULT_SCHEDULE_TIMEOUT
    assert computed.execution_timeout == DEFAULT_EXECUTION_TIMEOUT
    assert computed.retries == DEFAULT_RETRIES


def test_unset_values_fall_back_to_task_defaults() -> None:
    computed = compute_task_parameters(
        schedule_timeout=None,
        execution_timeout=None,
        retries=None,
        backoff_factor=None,
        backoff_max_seconds=None,
        task_defaults=TASK_DEFAULTS,
    )

    assert computed.schedule_timeout == TASK_DEFAULTS.schedule_timeout
    assert computed.execution_timeout == TASK_DEFAULTS.execution_timeout
    assert computed.retries == TASK_DEFAULTS.retries
    assert computed.backoff_factor == TASK_DEFAULTS.backoff_factor
    assert computed.backoff_max_seconds == TASK_DEFAULTS.backoff_max_seconds


def test_unset_values_fall_back_to_built_in_defaults_without_task_defaults() -> None:
    computed = compute_task_parameters(
        schedule_timeout=None,
        execution_timeout=None,
        retries=None,
        backoff_factor=None,
        backoff_max_seconds=None,
        task_defaults=TaskDefaults(),
    )

    assert computed.schedule_timeout == DEFAULT_SCHEDULE_TIMEOUT
    assert computed.execution_timeout == DEFAULT_EXECUTION_TIMEOUT
    assert computed.retries == DEFAULT_RETRIES
    assert computed.backoff_factor is None
    assert computed.backoff_max_seconds is None


def test_workflow_task_keeps_an_explicit_zero_retries(hatchet: Hatchet) -> None:
    workflow = hatchet.workflow(
        name="explicit-zero-retries", task_defaults=TASK_DEFAULTS
    )

    @workflow.task(retries=0, execution_timeout=timedelta(seconds=60))
    def explicit(input: None, ctx: Context) -> None:
        return None

    @workflow.task()
    def inherited(input: None, ctx: Context) -> None:
        return None

    assert explicit.retries == 0
    assert explicit.execution_timeout == timedelta(seconds=60)
    assert inherited.retries == TASK_DEFAULTS.retries
    assert inherited.execution_timeout == TASK_DEFAULTS.execution_timeout


@pytest.mark.parametrize(
    "concurrency",
    [
        CONCURRENCY_STRATEGY,
        [CONCURRENCY_STRATEGY],
        (CONCURRENCY_STRATEGY,),
    ],
)
def test_concurrency_is_normalized_to_a_list_of_strategies(concurrency: Any) -> None:
    assert normalize_concurrency_to_strategies(concurrency) == [CONCURRENCY_STRATEGY]


def test_integer_and_missing_concurrency_are_normalized() -> None:
    assert normalize_concurrency_to_strategies(None) == []
    assert normalize_concurrency_to_strategies(3) == [ConcurrencyStrategy.from_int(3)]


def test_standalone_task_keeps_a_tuple_of_concurrency_strategies(
    hatchet: Hatchet,
) -> None:
    @hatchet.task(concurrency=(CONCURRENCY_STRATEGY,))
    def with_tuple_concurrency(input: None, ctx: Context) -> None:
        return None

    assert with_tuple_concurrency._task.concurrency == [CONCURRENCY_STRATEGY]
