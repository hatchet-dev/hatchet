import asyncio
from datetime import datetime, timedelta
from typing import Literal
from uuid import uuid4

import pytest
from pydantic import BaseModel

from examples.concurrency_multiple_keys.worker import (
    SLEEP_TIME,
    WorkflowInput,
    concurrency_multiple_keys_workflow,
)
from hatchet_sdk import Hatchet, TriggerWorkflowOptions

Character = Literal["Anna", "Vronsky", "Stiva", "Dolly", "Levin", "Karenin"]
characters: list[Character] = [
    "Anna",
    "Vronsky",
    "Stiva",
    "Dolly",
    "Levin",
    "Karenin",
]


class AdditionalMetadata(BaseModel):
    test_run_id: str
    key: str
    name: Character
    digit: str


@pytest.mark.asyncio()
async def test_priority(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
    tasks: list[tuple[Character, str]] = [
        ("Anna", "1"),
        ("Anna", "2"),
        ("Vronsky", "1"),
        ("Stiva", "1"),
    ]

    run_refs = await concurrency_multiple_keys_workflow.aio_run_many_no_wait(
        [
            concurrency_multiple_keys_workflow.create_bulk_run_item(
                WorkflowInput(
                    name=name,
                    digit=digit,
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
            for name, digit in tasks
        ]
    )

    await asyncio.gather(*[r.aio_result() for r in run_refs])
    await asyncio.sleep(3)

    workflows = (
        await hatchet.workflows.aio_list(
            workflow_name=concurrency_multiple_keys_workflow.name
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
    )

    sorted_runs = sorted(runs.rows, key=lambda r: r.started_at or datetime.min)

    first_task_started_at = sorted_runs[0].started_at or datetime.max
    first_three_runs = sorted_runs[:3]

    last_run = sorted_runs[-1]
    first_anna_run = next(
        (
            r
            for r in first_three_runs
            if AdditionalMetadata.model_validate(r.additional_metadata).name == "Anna"
        ),
    )

    """The two Anna runs should not be able to run concurrently, but everything else can"""
    assert {
        AdditionalMetadata.model_validate(r.additional_metadata).name
        for r in first_three_runs
    } == {
        "Anna",
        "Vronsky",
        "Stiva",
    }

    """The second Anna run should start after the first Anna run finishes"""
    assert (
        AdditionalMetadata.model_validate(last_run.additional_metadata).name == "Anna"
    )
    assert (last_run.started_at or datetime.min) > (
        first_anna_run.finished_at or datetime.max
    )
    assert (last_run.started_at or datetime.min) > first_task_started_at + timedelta(
        seconds=SLEEP_TIME
    )
