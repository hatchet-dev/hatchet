import asyncio

from agents import Agent, Runner
from examples.agent.workflows import create_temperature_workflow_tool_openai


async def main() -> None:
    temperature_tool = create_temperature_workflow_tool_openai()

    agent = Agent(
        name="Assistant",
        tools=[temperature_tool],
    )
    result = await Runner.run(agent, "What's the temperature in San Francisco?")
    print(result.final_output)


if __name__ == "__main__":
    asyncio.run(main())
