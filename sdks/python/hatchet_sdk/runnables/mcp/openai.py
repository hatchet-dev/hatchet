import json
import typing
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from hatchet_sdk.runnables.types import R
    from hatchet_sdk.runnables.workflow import Standalone, Workflow

try:
    from agents import FunctionTool
    from agents.tool_context import ToolContext
except (RuntimeError, ImportError, ModuleNotFoundError) as e:
    raise ModuleNotFoundError(
        "To use the mcp_tool method with OpenAI as a provider, you must install Hatchet's `openai` extra using (e.g.) `pip install hatchet-sdk[openai]`"
    ) from e


def task_to_openai_mcp(
    runnable: "Standalone[Any, R]",
    input_schema: dict[str, Any],
    description: str,
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
    )


def workflow_to_openai_mcp(
    runnable: "Workflow[Any]", input_schema: dict[str, Any], description: str
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
    )
