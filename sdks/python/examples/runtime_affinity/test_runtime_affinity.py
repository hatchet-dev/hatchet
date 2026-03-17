import pytest
from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from hatchet_sdk.labels import DesiredWorkerLabel
from subprocess import Popen
from typing import Any, Generator
from examples.runtime_affinity.worker import affinity_example_task
from random import choice
from conftest import _on_demand_worker_fixture

labels = ["foo", "bar"]


@pytest.fixture(scope="session")
def on_demand_worker_a(
    request: pytest.FixtureRequest,
) -> Generator[Popen[bytes], None, None]:
    yield from _on_demand_worker_fixture(request)


@pytest.fixture(scope="session")
def on_demand_worker_b(
    request: pytest.FixtureRequest,
) -> Generator[Popen[bytes], None, None]:
    yield from _on_demand_worker_fixture(request)


@pytest.mark.parametrize(
    "on_demand_worker_a",
    [
        (
            [
                "poetry",
                "run",
                "python",
                "examples/runtime_affinity/worker.py",
                "--label",
                labels[0],
            ],
            8003,
        )
    ],
    indirect=True,
)
@pytest.mark.parametrize(
    "on_demand_worker_b",
    [
        (
            [
                "poetry",
                "run",
                "python",
                "examples/runtime_affinity/worker.py",
                "--label",
                labels[1],
            ],
            8004,
        )
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_runtime_affinity(
    hatchet: Hatchet,
    on_demand_worker_a: Popen[Any],
    on_demand_worker_b: Popen[Any],
) -> None:
    workers = [
        w
        for w in (await hatchet.workers.aio_list()).rows or []
        if w.status == "ACTIVE"
        and w.name == hatchet.config.apply_namespace("runtime-affinity-worker")
    ]

    assert len(workers) == 2

    worker_label_to_id = {
        label.value: worker.metadata.id
        for worker in workers
        for label in (worker.labels or [])
        if label.key == "affinity" and label.value in labels
    }

    assert set(worker_label_to_id.keys()) == set(labels)

    for _ in range(20):
        target_worker = choice(labels)
        res = await affinity_example_task.aio_run(
            options=TriggerWorkflowOptions(
                desired_worker_label=[
                    DesiredWorkerLabel(
                        key="affinity",
                        value=target_worker,
                        required=True,
                    ),
                ]
            )
        )
        assert res.worker_id == worker_label_to_id[target_worker]
