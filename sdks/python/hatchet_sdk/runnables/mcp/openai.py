from __future__ import annotations

import json
import typing
from collections.abc import Sequence
from typing import TYPE_CHECKING, Any

from hatchet_sdk.runnables.workflow import Standalone, Workflow

if TYPE_CHECKING:
    from hatchet_sdk.runnables.types import R

try:
    from agents import FunctionTool, Tool
    from agents.tool_context import ToolContext
except (RuntimeError, ImportError, ModuleNotFoundError) as e:
    raise ModuleNotFoundError(
        "To use OpenAI tools, install Hatchet's `openai` extra: `pip install hatchet-sdk[openai]`"
    ) from e

Runnable = Workflow[Any] | Standalone[Any, Any]


def to_openai_tools(runnables: Sequence[Runnable]) -> list[Tool]:
    """Convert a list of Hatchet workflows/tasks into OpenAI ``FunctionTool`` instances.

    Descriptions are read from the workflow/task config (set at definition time).

    :param runnables: Hatchet workflows or standalone tasks to convert.
    :returns: A list of OpenAI tools ready to pass to ``Agent(tools=...)``.
    :raises ValueError: If any runnable has no description set.
    """
    tools: list[Tool] = []
    for r in runnables:
        description = r.config.description
        if not description:
            raise ValueError(
                f"Runnable '{r.config.name}' has no description. "
                "Set description= when defining the workflow or task."
            )
        input_schema = r.input_validator.json_schema()
        if isinstance(r, Standalone):
            tools.append(_task_to_openai(r, input_schema, description))
        else:
            tools.append(_workflow_to_openai(r, input_schema, description))
    return tools


def _task_to_openai(
    runnable: Standalone[Any, R],
    input_schema: dict[str, Any],
    description: str,
    **kwargs: Any,
) -> FunctionTool:
    async def handler(ctx: ToolContext[Any], input: str) -> str:
        loaded_input = typing.cast(dict[str, Any], json.loads(input))
        res = await runnable.aio_run(loaded_input)
        return runnable.output_validator.dump_json(res).decode("utf-8")

    return FunctionTool(
        name=runnable.name,
        description=description,
        params_json_schema=input_schema,
        on_invoke_tool=handler,
        **kwargs,
    )


def _workflow_to_openai(
    runnable: Workflow[Any],
    input_schema: dict[str, Any],
    description: str,
    **kwargs: Any,
) -> FunctionTool:
    async def handler(ctx: ToolContext[Any], input: str) -> str:
        loaded_input = typing.cast(dict[str, Any], json.loads(input))
        res = await runnable.aio_run(loaded_input)
        return json.dumps(res)

    return FunctionTool(
        name=runnable.name,
        description=description,
        params_json_schema=input_schema,
        on_invoke_tool=handler,
        **kwargs,
    )
