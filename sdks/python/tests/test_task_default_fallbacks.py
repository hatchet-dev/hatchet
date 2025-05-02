"""
IMPORTANT
----------

These tests are intended to prevent us from changing defaults in one place and
forgetting to change them in other places, or otherwise breaking the default handling logic.
If you get a failure here, you likely changed the default values for some of the params (below)
in one of the task decorators e.g. `Workflow.task`, etc.

The intention of these tests is to:
 1. Ensure that the behavior of falling back to `TaskDefaults` works as expected, which means that
    if no value for a certain parameter to one of these decorators is provided, it should fall back to the
    value in `TaskDefaults` if one is set.
2. Ensure that the default values set in the rest of the codebase don't change, and are consistent with each other.

If you change the default values for any of these parameters, please update the tests accordingly.
"""

from datetime import timedelta
from typing import Any

import pytest

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet, Task, TaskDefaults


def dummy_task(input: EmptyModel, context: Context) -> dict[str, str]:
    return {"foo": "bar"}


def dummy_durable_task(input: EmptyModel, context: DurableContext) -> dict[str, str]:
    return {"foo": "bar"}


DEFAULT_SCHEDULE_TIMEOUT = timedelta(minutes=5)
DEFAULT_EXECUTION_TIMEOUT = timedelta(seconds=60)
DEFAULT_RETRIES = 0
DEFAULT_BACKOFF_FACTOR = None
DEFAULT_BACKOFF_MAX_SECONDS = None


def task(
    hatchet: Hatchet,
    is_durable: bool,
    task_defaults: TaskDefaults,
    **kwargs: Any,
) -> Task[EmptyModel, dict[str, str]]:
    workflow = hatchet.workflow(
        name="foo",
        task_defaults=task_defaults,
    )

    task_fn = workflow.durable_task if is_durable else workflow.task

    return task_fn(**kwargs)(dummy_durable_task if is_durable else dummy_task)  # type: ignore


def standalone_task(
    hatchet: Hatchet,
    is_durable: bool,
    **kwargs: Any,
) -> Task[EmptyModel, dict[str, str]]:
    task_fn = hatchet.durable_task if is_durable else hatchet.task

    return task_fn(**kwargs)(dummy_durable_task if is_durable else task)._task  # type: ignore


@pytest.mark.parametrize("is_durable", [False, True])
def test_task_defaults_applied_correctly(hatchet: Hatchet, is_durable: bool) -> None:
    schedule_timeout = timedelta(seconds=3)
    execution_timeout = timedelta(seconds=1)
    retries = 4
    backoff_factor = 1
    backoff_max_seconds = 5

    t = task(
        hatchet=hatchet,
        is_durable=is_durable,
        task_defaults=TaskDefaults(
            schedule_timeout=schedule_timeout,
            execution_timeout=execution_timeout,
            retries=retries,
            backoff_factor=backoff_factor,
            backoff_max_seconds=backoff_max_seconds,
        ),
    )

    assert t.schedule_timeout == schedule_timeout
    assert t.execution_timeout == execution_timeout
    assert t.retries == retries
    assert t.backoff_factor == backoff_factor
    assert t.backoff_max_seconds == backoff_max_seconds


@pytest.mark.parametrize(
    "is_durable,is_standalone",
    [
        (False, False),
        (True, False),
        (False, True),
        (True, True),
    ],
)
def test_fallbacking_ensure_default_unchanged(
    hatchet: Hatchet, is_durable: bool, is_standalone: bool
) -> None:
    t = task(
        hatchet=hatchet,
        is_durable=is_durable,
        task_defaults=TaskDefaults(),
    )

    """If this test fails, it means that you changed the default values for the params to one of the `task` or `durable_task` decorators"""
    assert t.schedule_timeout == DEFAULT_SCHEDULE_TIMEOUT
    assert t.execution_timeout == DEFAULT_EXECUTION_TIMEOUT
    assert t.retries == DEFAULT_RETRIES
    assert t.backoff_factor == DEFAULT_BACKOFF_FACTOR
    assert t.backoff_max_seconds == DEFAULT_BACKOFF_MAX_SECONDS

@pytest.mark.parametrize(
    "is_durable,is_standalone",
    [
        (False, False),
        (True, False),
        (False, True),
        (True, True),
    ],
)
def test_defaults_correctly_overridden_by_params_passed_in(
    hatchet: Hatchet, is_durable: bool, is_standalone: bool
) -> None:
    t = task(
        hatchet=hatchet,
        is_durable=is_durable,
        task_defaults=TaskDefaults(
            schedule_timeout=timedelta(seconds=3),
            execution_timeout=timedelta(seconds=1),
            retries=4,
            backoff_factor=1,
            backoff_max_seconds=5,
        ),
        schedule_timeout=timedelta(seconds=9),
        execution_timeout=timedelta(seconds=2),
        retries=6,
        backoff_factor=3,
        backoff_max_seconds=5,
    )

    """If this test fails, it means that you changed the default values for the params to the `task` method on the Workflow"""
    assert t.schedule_timeout == timedelta(seconds=9)
    assert t.execution_timeout == timedelta(seconds=2)
    assert t.retries == 6
    assert t.backoff_factor == 3
    assert t.backoff_max_seconds == 5

