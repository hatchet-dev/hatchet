import asyncio

from agents import Agent, Runner
from hatchet_sdk.runnables.mcp.openai import to_openai_tools
from examples.agent.worker import temp_workflow, get_temperature_standalone


async def main() -> None:
    tools = to_openai_tools([temp_workflow, get_temperature_standalone])

    agent = Agent(
        name="Assistant",
        tools=tools,
    )
    result = await Runner.run(agent, "What's the temperature in San Francisco?")
    print(result.final_output)


if __name__ == "__main__":
    asyncio.run(main())
