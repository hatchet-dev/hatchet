import asyncio

from agents import Agent, Runner
from examples.agent.workflows import temperature_tool_openai


async def main() -> None:
    agent = Agent(
        name="Assistant",
        tools=[temperature_tool_openai],
    )
    result = await Runner.run(agent, "What's the temperature in San Francisco?")
    print(result.final_output)


if __name__ == "__main__":
    asyncio.run(main())
