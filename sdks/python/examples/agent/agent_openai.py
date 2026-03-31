import asyncio

from agents import Agent, Runner
from hatchet_sdk.runnables.workflow import MCPProvider
from examples.agent.worker import temp_workflow, get_temperature_standalone


async def main() -> None:
    # You can use a workflow
    temperature_tool = temp_workflow.mcp_tool(
        MCPProvider.OPENAI,
        "Get the current temperature at a location",
    )

    # Or a standalone task
    temperature_tool = get_temperature_standalone.mcp_tool(
        MCPProvider.OPENAI,
        "Get the current temperature at a location",
    )

    agent = Agent(
        name="Assistant",
        tools=[temperature_tool],
    )
    result = await Runner.run(agent, "What's the temperature in San Francisco?")
    print(result)


if __name__ == "__main__":
    asyncio.run(main())
