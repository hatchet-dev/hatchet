from typing import Any

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()

parent_returns_none_dag = hatchet.workflow(name="dag-parent-returns-none")


@parent_returns_none_dag.task()
async def parent_returning_none(input: EmptyModel, ctx: Context) -> None:
    return None


@parent_returns_none_dag.task()
async def parent_returning_empty_dict(
    input: EmptyModel, ctx: Context
) -> dict[str, Any]:
    return {}


@parent_returns_none_dag.task(
    parents=[parent_returning_none, parent_returning_empty_dict]
)
async def child_reading_parent_outputs(
    input: EmptyModel, ctx: Context
) -> dict[str, Any]:
    return {
        "none_parent_output": ctx.task_output(parent_returning_none),
        "empty_dict_parent_output": ctx.task_output(parent_returning_empty_dict),
    }
