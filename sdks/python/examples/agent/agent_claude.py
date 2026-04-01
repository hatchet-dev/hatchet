import asyncio
from claude_agent_sdk import (
    create_sdk_mcp_server,
    ClaudeAgentOptions,
    query,
    ResultMessage,
)
from examples.agent.workflows import temperature_tool_claude


async def main() -> None:
    # Wrap the tool in an in-process MCP server
    weather_server = create_sdk_mcp_server(
        name="weather",
        version="1.0.0",
        tools=[temperature_tool_claude],
    )

    options = ClaudeAgentOptions(
        mcp_servers={"weather": weather_server},
        allowed_tools=[
            f"mcp__{weather_server["name"]}__{temperature_tool_claude.name}"
        ],
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
