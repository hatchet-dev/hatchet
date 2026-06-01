import asyncio

from claude_agent_sdk import (
    create_sdk_mcp_server,
    ClaudeAgentOptions,
    query,
    ResultMessage,
)
from examples.support_agent_tools.tools import (
    create_lookup_customer_tool_claude,
    create_check_order_status_tool_claude,
    create_ticket_tool_claude,
)


async def main() -> None:
    lookup_customer_tool = create_lookup_customer_tool_claude()
    check_order_status_tool = create_check_order_status_tool_claude()
    ticket_tool = create_ticket_tool_claude()

    support_server = create_sdk_mcp_server(
        name="support",
        version="1.0.0",
        tools=[lookup_customer_tool, check_order_status_tool, ticket_tool],
    )

    server_name = support_server["name"]
    options = ClaudeAgentOptions(
        mcp_servers={"support": support_server},
        allowed_tools=[
            f"mcp__{server_name}__{lookup_customer_tool.name}",
            f"mcp__{server_name}__{check_order_status_tool.name}",
            f"mcp__{server_name}__{ticket_tool.name}",
        ],
    )

    async for message in query(
        prompt=(
            "Customer C-100 says order ORD-9987 has not arrived. "
            "Look up the customer, check the order status, and create a "
            "support ticket if the order has a known issue or delayed delivery. "
            'If you create a ticket, use priority "high", subject '
            '"Delayed order ORD-9987", and a body that summarizes the known '
            "carrier delay. Then summarize what happened."
        ),
        options=options,
    ):
        print(message)
        if isinstance(message, ResultMessage) and message.subtype == "success":
            print(message.result)


if __name__ == "__main__":
    asyncio.run(main())
