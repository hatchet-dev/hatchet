import pytest

from hatchet_sdk.exceptions import FailedTaskRunExceptionGroup, TaskRunError


def test_empty_exception_list_falls_back_to_an_error_carrying_the_message() -> None:
    group = FailedTaskRunExceptionGroup("run failed", [])

    assert len(group.exceptions) == 1
    assert group.exceptions[0].exc == "run failed"


def test_exception_group_is_caught_by_except_star_and_not_by_value_error() -> None:
    error = TaskRunError(
        exc="boom", exc_type="RuntimeError", trace="", task_run_external_id="task-1"
    )

    with pytest.raises(ExceptionGroup):
        try:
            raise FailedTaskRunExceptionGroup("run failed", [error])
        except ValueError:
            pytest.fail("FailedTaskRunExceptionGroup should not be a ValueError")

    caught: list[TaskRunError] = []
    try:
        raise FailedTaskRunExceptionGroup("run failed", [error])
    except* TaskRunError as matched:
        caught.extend(matched.exceptions)  # type: ignore[arg-type]

    assert caught == [error]
