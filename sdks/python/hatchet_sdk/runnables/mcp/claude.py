import json
from typing import TYPE_CHECKING, Any

from mcp.types import ToolAnnotations

from hatchet_sdk.runnables.types import TWorkflowInput

if TYPE_CHECKING:
    from hatchet_sdk.runnables.types import R
    from hatchet_sdk.runnables.workflow import Standalone, Workflow
try:
    from claude_agent_sdk import SdkMcpTool
except (RuntimeError, ImportError, ModuleNotFoundError) as e:
    raise ModuleNotFoundError(
        "To use the mcp_tool method with Claude as a provider, you must install Hatchet's `claude` extra using (e.g.) `pip install hatchet-sdk[claude]`"
    ) from e


def task_to_claude_mcp(
    runnable: "Standalone[TWorkflowInput, R]",
    input_schema: dict[str, Any],
    description: str,
    annotations: ToolAnnotations | None = None,
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


def workflow_to_claude_mcp(
    runnable: "Workflow[TWorkflowInput]",
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
