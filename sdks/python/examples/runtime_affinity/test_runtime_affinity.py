import pytest
from hatchet_sdk import Hatchet
from hatchet_sdk.labels import DesiredWorkerLabel
from subprocess import Popen
from typing import Any, Generator
from examples.runtime_affinity.worker import runtime_affinity_workflow
from random import choice
from conftest import _on_demand_worker_fixture

labels = ["foo", "bar", "baz", "qux"]


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


@pytest.fixture(scope="session")
def on_demand_worker_c(
    request: pytest.FixtureRequest,
) -> Generator[Popen[bytes], None, None]:
    yield from _on_demand_worker_fixture(request)


@pytest.fixture(scope="session")
def on_demand_worker_d(
    request: pytest.FixtureRequest,
) -> Generator[Popen[bytes], None, None]:
    yield from _on_demand_worker_fixture(request)


@pytest.mark.parametrize(
    "on_demand_worker_a",
    [
        [
            "poetry",
            "run",
            "python",
            "examples/runtime_affinity/worker.py",
            "--label",
            labels[0],
        ]
    ],
    indirect=True,
)
@pytest.mark.parametrize(
    "on_demand_worker_b",
    [
        [
            "poetry",
            "run",
            "python",
            "examples/runtime_affinity/worker.py",
            "--label",
            labels[1],
        ]
    ],
    indirect=True,
)
@pytest.mark.parametrize(
    "on_demand_worker_c",
    [
        [
            "poetry",
            "run",
            "python",
            "examples/runtime_affinity/worker.py",
            "--label",
            labels[2],
        ]
    ],
    indirect=True,
)
@pytest.mark.parametrize(
    "on_demand_worker_d",
    [
        [
            "poetry",
            "run",
            "python",
            "examples/runtime_affinity/worker.py",
            "--label",
            labels[3],
        ]
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_runtime_affinity(
    hatchet: Hatchet,
    on_demand_worker_a: Popen[Any],
    on_demand_worker_b: Popen[Any],
    on_demand_worker_c: Popen[Any],
    on_demand_worker_d: Popen[Any],
) -> None:
    workers = [
        w
        for w in (await hatchet.workers.aio_list()).rows or []
        if w.status == "ACTIVE"
        and w.name == hatchet.config.apply_namespace("runtime-affinity-worker")
    ]

    assert len(workers) == 4

    worker_label_to_id = {
        label.value: worker.metadata.id
        for worker in workers
        for label in (worker.labels or [])
        if label.key == "affinity" and label.value in labels
    }

    assert set(worker_label_to_id.keys()) == set(labels)

    expected_tasks = [t.name for t in runtime_affinity_workflow.tasks]

    for _ in range(50):
        target_worker = choice(labels)
        res = await runtime_affinity_workflow.aio_run(
            desired_worker_labels=[
                DesiredWorkerLabel(
                    key="affinity",
                    value=target_worker,
                    required=True,
                ),
            ],
        )

        expected_worker_id = worker_label_to_id[target_worker]

        for task_name in expected_tasks:
            assert task_name in res, f"Task {task_name} not found in workflow result"
            assert res[task_name]["worker_id"] == expected_worker_id, (
                f"Task {task_name} ran on wrong worker. Expected {expected_worker_id}, got {res[task_name]['worker_id']}"
            )
