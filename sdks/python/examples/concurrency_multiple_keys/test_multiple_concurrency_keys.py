import asyncio
from collections import Counter
from datetime import datetime
from random import choice
from typing import Literal
from uuid import uuid4

import pytest
from pydantic import BaseModel

from examples.concurrency_multiple_keys.worker import (
    DIGIT_MAX_RUNS,
    NAME_MAX_RUNS,
    WorkflowInput,
    concurrency_multiple_keys_workflow,
)
from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary

Character = Literal["Anna", "Vronsky", "Stiva", "Dolly", "Levin", "Karenin"]
characters: list[Character] = [
    "Anna",
    "Vronsky",
    "Stiva",
    "Dolly",
    "Levin",
    "Karenin",
]


class RunMetadata(BaseModel):
    test_run_id: str
    key: str
    name: Character
    digit: str
    started_at: datetime
    finished_at: datetime

    @staticmethod
    def parse(task: V1TaskSummary) -> "RunMetadata":
        return RunMetadata(
            test_run_id=task.additional_metadata["test_run_id"],  # type: ignore
            key=task.additional_metadata["key"],  # type: ignore
            name=task.additional_metadata["name"],  # type: ignore
            digit=task.additional_metadata["digit"],  # type: ignore
            started_at=task.started_at or datetime.max,
            finished_at=task.finished_at or datetime.min,
        )

    def __str__(self) -> str:
        return self.key


@pytest.mark.asyncio()
async def test_multi_concurrency_key(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())

    run_refs = await concurrency_multiple_keys_workflow.aio_run_many_no_wait(
        [
            concurrency_multiple_keys_workflow.create_bulk_run_item(
                WorkflowInput(
                    name=(name := choice(characters)),
                    digit=(digit := choice([str(i) for i in range(6)])),
                ),
                options=TriggerWorkflowOptions(
                    additional_metadata={
                        "test_run_id": test_run_id,
                        "key": f"{name}-{digit}",
                        "name": name,
                        "digit": digit,
                    },
                ),
            )
            for _ in range(200)
        ]
    )

    await asyncio.gather(*[r.aio_result() for r in run_refs])

    workflows = (
        await hatchet.workflows.aio_list(
            workflow_name=concurrency_multiple_keys_workflow.name,
            limit=1_000,
        )
    ).rows

    assert workflows

    workflow = next(
        (w for w in workflows if w.name == concurrency_multiple_keys_workflow.name),
        None,
    )

    assert workflow

    assert workflow.name == concurrency_multiple_keys_workflow.name

    runs = await hatchet.runs.aio_list(
        workflow_ids=[workflow.metadata.id],
        additional_metadata={
            "test_run_id": test_run_id,
        },
        limit=1_000,
    )

    sorted_runs = sorted(
        [RunMetadata.parse(r) for r in runs.rows], key=lambda r: r.started_at
    )

    overlapping_groups: dict[int, list[RunMetadata]] = {}

    for run in sorted_runs:
        has_group_membership = False

        if not overlapping_groups:
            overlapping_groups[1] = [run]
            continue

        if has_group_membership:
            continue

        for id, group in overlapping_groups.items():
            if all(are_overlapping(run, task) for task in group):
                overlapping_groups[id].append(run)
                has_group_membership = True
                break

        if not has_group_membership:
            overlapping_groups[len(overlapping_groups) + 1] = [run]

    for id, group in overlapping_groups.items():
        assert is_valid_group(group), f"Group {id} is not valid"


def are_overlapping(x: RunMetadata, y: RunMetadata) -> bool:
    return (x.started_at < y.finished_at and x.finished_at > y.started_at) or (
        x.finished_at > y.started_at and x.started_at < y.finished_at
    )


def is_valid_group(group: list[RunMetadata]) -> bool:
    digits = Counter[str]()
    names = Counter[str]()

    for task in group:
        digits[task.digit] += 1
        names[task.name] += 1

    if any(v > DIGIT_MAX_RUNS for v in digits.values()):
        return False

    if any(v > NAME_MAX_RUNS for v in names.values()):
        return False

    return True
