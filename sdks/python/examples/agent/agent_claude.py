import asyncio
from claude_agent_sdk import (
    create_sdk_mcp_server,
    ClaudeAgentOptions,
    query,
    ResultMessage,
)
from hatchet_sdk.runnables.workflow import MCPProvider
from examples.agent.worker import temp_workflow, get_temperature_standalone


async def main() -> None:
    # You can use a workflow
    temperature_tool = temp_workflow.mcp_tool(
        MCPProvider.CLAUDE,
        "Get the current temperature at a location",
    )

    # Or a standalone task
    temperature_tool = get_temperature_standalone.mcp_tool(
        MCPProvider.CLAUDE,
        "Get the current temperature at a location",
    )

    # Wrap the tool in an in-process MCP server
    weather_server = create_sdk_mcp_server(
        name="weather",
        version="1.0.0",
        tools=[temperature_tool],
    )

    options = ClaudeAgentOptions(
        mcp_servers={"weather": weather_server},
        allowed_tools=[f"mcp__{weather_server["name"]}__{temperature_tool.name}"],
    )

    async for message in query(
        prompt="What's the temperature in San Francisco?",
        options=options,
    ):
        print(message)
        # ResultMessage is the final message after all tool calls complete
        if isinstance(message, ResultMessage) and message.subtype == "success":
            print(message.result)


if __name__ == "__main__":
    asyncio.run(main())
