import asyncio

from agents import Agent, Runner
from examples.support_agent_tools.tools import (
    create_lookup_customer_tool_openai,
    create_check_order_status_tool_openai,
    create_ticket_tool_openai,
)


async def main() -> None:
    lookup_customer_tool = create_lookup_customer_tool_openai()
    check_order_status_tool = create_check_order_status_tool_openai()
    ticket_tool = create_ticket_tool_openai()

    agent = Agent(
        name="support-agent",
        tools=[lookup_customer_tool, check_order_status_tool, ticket_tool],
    )

    result = await Runner.run(
        agent,
        "Customer C-100 says order ORD-9987 has not arrived. "
        "Look up the customer, check the order status, and create a "
        "support ticket if the order has a known issue or delayed delivery. "
        'If you create a ticket, use priority "high", subject '
        '"Delayed order ORD-9987", and a body that summarizes the known '
        "carrier delay. Then summarize what happened.",
    )
    print(result.final_output)


if __name__ == "__main__":
    asyncio.run(main())
