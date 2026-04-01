from __future__ import annotations

import json
from collections.abc import Sequence
from typing import TYPE_CHECKING, Any

from hatchet_sdk.runnables.types import TWorkflowInput
from hatchet_sdk.runnables.workflow import Standalone, Workflow

if TYPE_CHECKING:
    from hatchet_sdk.runnables.types import R

try:
    from claude_agent_sdk import SdkMcpTool
except (RuntimeError, ImportError, ModuleNotFoundError) as e:
    raise ModuleNotFoundError(
        "To use Claude MCP tools, install Hatchet's `claude` extra: `pip install hatchet-sdk[claude]`"
    ) from e

Runnable = Workflow[Any] | Standalone[Any, Any]


def to_claude_mcp_tools(
    runnables: Sequence[Runnable],
    server_name: str,
) -> tuple[list[SdkMcpTool[Any]], list[str]]:
    """Convert a list of Hatchet workflows/tasks into Claude ``SdkMcpTool`` instances.

    Descriptions are read from the workflow/task config (set at definition time).

    :param runnables: Hatchet workflows or standalone tasks to convert.
    :param server_name: MCP server name, used to build ``allowed_tools`` strings.
    :returns: A tuple of (tools, allowed_tool_names).
    :raises ValueError: If any runnable has no description set.
    """
    tools: list[SdkMcpTool[Any]] = []
    for r in runnables:
        description = r.config.description
        if not description:
            raise ValueError(
                f"Runnable '{r.config.name}' has no description. "
                "Set description= when defining the workflow or task."
            )
        input_schema = r.input_validator.json_schema()
        if isinstance(r, Standalone):
            tools.append(_task_to_claude(r, input_schema, description))
        else:
            tools.append(_workflow_to_claude(r, input_schema, description))

    tool_names = [f"mcp__{server_name}__{t.name}" for t in tools]
    return tools, tool_names


def _task_to_claude(
    runnable: Standalone[TWorkflowInput, R],
    input_schema: dict[str, Any],
    description: str,
    **kwargs: Any,
) -> SdkMcpTool[TWorkflowInput]:
    async def handler(input: TWorkflowInput) -> dict[str, Any]:
        res = await runnable.aio_run(input)
        return {
            "content": [
                {
                    "type": "text",
                    "text": runnable.output_validator.dump_json(res).decode("utf-8"),
                }
            ]
        }

    return SdkMcpTool(
        name=runnable.name,
        description=description,
        input_schema=input_schema,
        handler=handler,
        **kwargs,
    )


def _workflow_to_claude(
    runnable: Workflow[TWorkflowInput],
    input_schema: dict[str, Any],
    description: str,
    **kwargs: Any,
) -> SdkMcpTool[TWorkflowInput]:
    async def handler(input: TWorkflowInput) -> dict[str, Any]:
        res = await runnable.aio_run(input)
        return {"content": [{"type": "text", "text": json.dumps(res)}]}

    return SdkMcpTool(
        name=runnable.name,
        description=description,
        input_schema=input_schema,
        handler=handler,
        **kwargs,
    )
