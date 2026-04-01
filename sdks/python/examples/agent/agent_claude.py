import asyncio
from claude_agent_sdk import (
    create_sdk_mcp_server,
    ClaudeAgentOptions,
    query,
    ResultMessage,
)
from hatchet_sdk.runnables.mcp.claude import to_claude_mcp_tools
from examples.agent.worker import temp_workflow, get_temperature_standalone


async def main() -> None:
    # Convert the workflows/tasks into Claude MCP tools
    server_name = "weather"
    tools, tool_names = to_claude_mcp_tools(
        [temp_workflow, get_temperature_standalone],
        server_name=server_name,
    )

    # Wrap the tools in an in-process MCP server
    weather_server = create_sdk_mcp_server(
        name=server_name,
        version="1.0.0",
        tools=tools,
    )

    options = ClaudeAgentOptions(
        mcp_servers={"weather": weather_server},
        allowed_tools=tool_names,
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
