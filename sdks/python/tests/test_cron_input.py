"""Unit tests for the `cron_input` option on workflow and task declarations.

`cron_input` lets you declare an input that is passed to runs triggered by a
workflow's `on_crons` schedules. These tests assert that the value is serialized
onto the `cron_input` field of the `CreateWorkflowVersionRequest` proto, and that
the field is left unset when no `cron_input` is provided.
"""

import json

from pydantic import BaseModel

from hatchet_sdk import Context, DurableContext, Hatchet


class CronInput(BaseModel):
    name: str
    count: int


def test_cron_input_serialized_onto_workflow_proto(hatchet: Hatchet) -> None:
    workflow = hatchet.workflow(
        name="cron-input-workflow",
        input_validator=CronInput,
        on_crons=["* * * * *"],
        cron_input=CronInput(name="alice", count=3),
    )

    @workflow.task()
    def step(input: CronInput, ctx: Context) -> dict[str, str]:
        return {"name": input.name}

    proto = workflow.to_proto()

    assert proto.HasField("cron_input")
    assert json.loads(proto.cron_input) == {"name": "alice", "count": 3}


def test_cron_input_unset_when_not_provided(hatchet: Hatchet) -> None:
    workflow = hatchet.workflow(
        name="no-cron-input-workflow",
        on_crons=["* * * * *"],
    )

    @workflow.task()
    def step(input: None, ctx: Context) -> dict[str, str]:
        return {}

    assert not workflow.to_proto().HasField("cron_input")


def test_cron_input_serialized_on_standalone_task(hatchet: Hatchet) -> None:
    @hatchet.task(
        name="cron-input-task",
        input_validator=CronInput,
        on_crons=["* * * * *"],
        cron_input=CronInput(name="bob", count=1),
    )
    def my_task(input: CronInput, ctx: Context) -> dict[str, str]:
        return {"name": input.name}

    proto = my_task.to_proto()

    assert proto.HasField("cron_input")
    assert json.loads(proto.cron_input) == {"name": "bob", "count": 1}


def test_cron_input_serialized_on_durable_task(hatchet: Hatchet) -> None:
    @hatchet.durable_task(
        name="cron-input-durable-task",
        input_validator=CronInput,
        on_crons=["* * * * *"],
        cron_input=CronInput(name="carol", count=2),
    )
    async def my_durable_task(input: CronInput, ctx: DurableContext) -> dict[str, str]:
        return {"name": input.name}

    proto = my_durable_task.to_proto()

    assert proto.HasField("cron_input")
    assert json.loads(proto.cron_input) == {"name": "carol", "count": 2}
