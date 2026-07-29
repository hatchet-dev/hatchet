import pytest

from examples.bug_tests.durable_spawn_index_collision.worker import (
    SpawnIndexCollisionInput,
    durable_spawn_index_collision,
)
from hatchet_sdk import Hatchet, V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_spawn_index_collision_fails_loudly(hatchet: Hatchet) -> None:
    ref = await durable_spawn_index_collision.aio_run(
        input=SpawnIndexCollisionInput(scenario="collision"), wait_for_result=False
    )

    with pytest.raises(Exception, match="[nN]on-determinism"):
        await ref.aio_result()

    runs = await hatchet.runs.aio_list(parent_task_external_id=ref.workflow_run_id)

    assert len(runs.rows) == 2, "the colliding spawn must not create a third child"


@pytest.mark.asyncio(loop_scope="session")
async def test_spawn_index_self_dedupe_returns_cached_result(hatchet: Hatchet) -> None:
    ref = await durable_spawn_index_collision.aio_run(
        input=SpawnIndexCollisionInput(scenario="self_dedupe"), wait_for_result=False
    )

    result = await ref.aio_result()

    assert result["first"] == {"which_child": "a"}
    assert result["respawned"] == {"which_child": "a"}

    runs = await hatchet.runs.aio_list(parent_task_external_id=ref.workflow_run_id)

    assert len(runs.rows) == 2, "the deduped spawn must not create a third child"
    assert all(run.status == V1TaskStatus.COMPLETED for run in runs.rows)
