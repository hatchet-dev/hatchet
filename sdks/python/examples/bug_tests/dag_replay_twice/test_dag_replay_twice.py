import asyncio
from typing import Any

import pytest

from examples.bug_tests.dag_replay_twice.worker import (
    DagReplayTwiceInput,
    dag_replay_twice,
    first,
    second,
)
from examples.test_utils import wait_for_running_status
from hatchet_sdk import FailedTaskRunExceptionGroup, Hatchet, RunStatus

# The DAG operator reports that it is blocked every 10 seconds, and the engine evicts the idle
# invocation on that report. A step that runs longer than that forces the operator to be
# evicted and then restored (re-invoked) once the step finishes, which replays the DAG from
# the top on a fresh invocation.
SECOND_DURATION_FORCING_OPERATOR_EVICTION_SECONDS = 15


async def wait_for_task_status(
    hatchet: Hatchet,
    run_id: str,
    task_name: str,
    status: RunStatus,
    timeout: float = 60.0,
) -> None:
    interval = 0.5

    for _ in range(int(timeout / interval)):
        details = await hatchet.runs.aio_get_details(run_id)
        task_run = details.task_runs.get(task_name)

        if task_run is not None and task_run.status == status:
            return

        await asyncio.sleep(interval)

    raise TimeoutError(f"task {task_name} of run {run_id} did not reach {status}")


async def wait_for_run_status(
    hatchet: Hatchet, run_id: str, status: RunStatus, timeout: float = 60.0
) -> None:
    interval = 0.5

    for _ in range(int(timeout / interval)):
        details = await hatchet.runs.aio_get_details(run_id)

        if details.status == status:
            return

        await asyncio.sleep(interval)

    raise TimeoutError(f"run {run_id} did not reach {status}")


async def run_until_first_step_fails(
    hatchet: Hatchet, second_duration_seconds: float = 1
) -> str:
    ref = await dag_replay_twice.aio_run(
        DagReplayTwiceInput(second_duration_seconds=second_duration_seconds),
        wait_for_result=False,
    )

    with pytest.raises(FailedTaskRunExceptionGroup):
        await ref.aio_result()

    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)

    assert details.task_runs[first.name].status == RunStatus.FAILED
    assert details.task_runs[second.name].status == RunStatus.CANCELLED

    return ref.workflow_run_id


async def replay_and_await_completion(
    hatchet: Hatchet, run_id: str, replay_target_id: str
) -> dict[str, dict[str, Any] | None]:
    await hatchet.runs.aio_replay(replay_target_id)
    await wait_for_running_status(hatchet, run_id)
    await wait_for_run_status(hatchet, run_id, RunStatus.COMPLETED, timeout=90)

    details = await hatchet.runs.aio_get_details(run_id)
    assert details.task_runs[first.name].status == RunStatus.COMPLETED
    assert details.task_runs[second.name].status == RunStatus.COMPLETED

    return {name: task_run.output for name, task_run in details.task_runs.items()}


@pytest.mark.timeout(120, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_dag_replayed_after_a_single_task_replay(hatchet: Hatchet) -> None:
    run_id = await run_until_first_step_fails(hatchet)
    details = await hatchet.runs.aio_get_details(run_id)

    result = await replay_and_await_completion(
        hatchet, run_id, details.task_runs[first.name].external_id
    )
    assert result[first.name] == {"retry_count": 1}
    assert result[second.name] == {"retry_count": 1}

    result = await replay_and_await_completion(hatchet, run_id, run_id)
    assert result[first.name] == {"retry_count": 2}
    assert result[second.name] == {"retry_count": 2}


@pytest.mark.timeout(120, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_dag_completes_when_operator_is_restored_after_a_single_task_replay(
    hatchet: Hatchet,
) -> None:
    run_id = await run_until_first_step_fails(
        hatchet, SECOND_DURATION_FORCING_OPERATOR_EVICTION_SECONDS
    )
    details = await hatchet.runs.aio_get_details(run_id)

    result = await replay_and_await_completion(
        hatchet, run_id, details.task_runs[first.name].external_id
    )
    assert result[first.name] == {"retry_count": 1}
    assert result[second.name] == {"retry_count": 1}


@pytest.mark.timeout(120, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_dag_whole_run_replay_then_single_task_replay(hatchet: Hatchet) -> None:
    run_id = await run_until_first_step_fails(hatchet)

    result = await replay_and_await_completion(hatchet, run_id, run_id)
    assert result[first.name] == {"retry_count": 1}
    assert result[second.name] == {"retry_count": 1}

    details = await hatchet.runs.aio_get_details(run_id)
    result = await replay_and_await_completion(
        hatchet, run_id, details.task_runs[first.name].external_id
    )
    assert result[first.name] == {"retry_count": 2}
    assert result[second.name] == {"retry_count": 2}


@pytest.mark.timeout(180, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_dag_single_task_replay_then_cancel_then_whole_run_replay(
    hatchet: Hatchet,
) -> None:
    run_id = await run_until_first_step_fails(
        hatchet, SECOND_DURATION_FORCING_OPERATOR_EVICTION_SECONDS
    )
    details = await hatchet.runs.aio_get_details(run_id)

    await hatchet.runs.aio_replay(details.task_runs[first.name].external_id)
    await wait_for_task_status(hatchet, run_id, second.name, RunStatus.RUNNING)

    await hatchet.runs.aio_cancel(run_id)
    await wait_for_run_status(hatchet, run_id, RunStatus.CANCELLED)

    details = await hatchet.runs.aio_get_details(run_id)
    assert details.task_runs[first.name].status == RunStatus.COMPLETED
    assert details.task_runs[second.name].status == RunStatus.CANCELLED

    result = await replay_and_await_completion(hatchet, run_id, run_id)
    assert result[first.name] == {"retry_count": 2}
    assert result[second.name] == {"retry_count": 2}


@pytest.mark.timeout(180, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_dag_completes_after_replaying_a_step_that_is_still_running(
    hatchet: Hatchet,
) -> None:
    ref = await dag_replay_twice.aio_run(
        DagReplayTwiceInput(
            first_fails_on_first_attempt=False,
            first_duration_seconds=SECOND_DURATION_FORCING_OPERATOR_EVICTION_SECONDS,
        ),
        wait_for_result=False,
    )
    run_id = ref.workflow_run_id

    await wait_for_task_status(hatchet, run_id, first.name, RunStatus.RUNNING)

    details = await hatchet.runs.aio_get_details(run_id)
    await hatchet.runs.aio_replay(details.task_runs[first.name].external_id)

    await wait_for_run_status(hatchet, run_id, RunStatus.COMPLETED, timeout=120)

    details = await hatchet.runs.aio_get_details(run_id)
    assert details.task_runs[first.name].status == RunStatus.COMPLETED
    assert details.task_runs[second.name].status == RunStatus.COMPLETED
