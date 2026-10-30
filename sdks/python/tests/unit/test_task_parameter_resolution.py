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
    backoff_max=timedelta(seconds=30),
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
        backoff_max=None,
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
        backoff_max=None,
        task_defaults=TASK_DEFAULTS,
    )

    assert computed.schedule_timeout == TASK_DEFAULTS.schedule_timeout
    assert computed.execution_timeout == TASK_DEFAULTS.execution_timeout
    assert computed.retries == TASK_DEFAULTS.retries
    assert computed.backoff_factor == TASK_DEFAULTS.backoff_factor
    assert computed.backoff_max == TASK_DEFAULTS.backoff_max


def test_unset_values_fall_back_to_built_in_defaults_without_task_defaults() -> None:
    computed = compute_task_parameters(
        schedule_timeout=None,
        execution_timeout=None,
        retries=None,
        backoff_factor=None,
        backoff_max=None,
        task_defaults=TaskDefaults(),
    )

    assert computed.schedule_timeout == DEFAULT_SCHEDULE_TIMEOUT
    assert computed.execution_timeout == DEFAULT_EXECUTION_TIMEOUT
    assert computed.retries == DEFAULT_RETRIES
    assert computed.backoff_factor is None
    assert computed.backoff_max is None


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

    assert explicit._retries == 0
    assert explicit._execution_timeout == timedelta(seconds=60)
    assert inherited._retries == TASK_DEFAULTS.retries
    assert inherited._execution_timeout == TASK_DEFAULTS.execution_timeout


def test_on_failure_task_falls_back_to_workflow_task_defaults(hatchet: Hatchet) -> None:
    workflow = hatchet.workflow(
        name="on-failure-task-defaults", task_defaults=TASK_DEFAULTS
    )

    @workflow.on_failure_task()
    def on_failure(input: None, ctx: Context) -> None:
        return None

    assert on_failure._schedule_timeout == TASK_DEFAULTS.schedule_timeout
    assert on_failure._execution_timeout == TASK_DEFAULTS.execution_timeout
    assert on_failure._retries == TASK_DEFAULTS.retries
    assert on_failure._backoff_factor == TASK_DEFAULTS.backoff_factor
    assert on_failure._backoff_max == TASK_DEFAULTS.backoff_max


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

    assert with_tuple_concurrency._task._concurrency == [CONCURRENCY_STRATEGY]
