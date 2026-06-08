# > Setup
from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from agents import FunctionTool
    from claude_agent_sdk import SdkMcpTool

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.runnables.workflow import MCPProvider

hatchet = Hatchet(debug=True)
# !!


# > Models
class CustomerLookupInput(BaseModel):
    customer_id: str


class CustomerInfo(BaseModel):
    customer_id: str
    name: str
    email: str
    plan: str
    account_status: str
    default_order_id: str
    support_tier: str


class OrderStatusInput(BaseModel):
    order_id: str


class OrderStatus(BaseModel):
    order_id: str
    status: str
    last_updated: str
    estimated_delivery: str
    known_issue: str | None
    carrier: str
    tracking_number: str


class CreateTicketInput(BaseModel):
    customer_id: str
    order_id: str
    subject: str
    body: str
    priority: str


class TicketResult(BaseModel):
    ticket_id: str
    status: str
    priority: str
    routing_team: str
    summary: str


# !!


# > Fixture data
CUSTOMERS = {
    "C-100": CustomerInfo(
        customer_id="C-100",
        name="Alice Martin",
        email="alice@example.com",
        plan="business",
        account_status="active",
        default_order_id="ORD-9987",
        support_tier="priority",
    ),
}

ORDERS = {
    "ORD-9987": OrderStatus(
        order_id="ORD-9987",
        status="delayed",
        last_updated="2026-05-20T14:30:00Z",
        estimated_delivery="2026-05-28",
        known_issue="Carrier reported weather delay at regional hub",
        carrier="FastShip",
        tracking_number="FS-482910",
    ),
}
# !!


# > Lookup customer
@hatchet.task(
    name="lookup-customer",
    input_validator=CustomerLookupInput,
    description="Look up a customer by ID and return their profile, plan, and support tier.",
)
async def lookup_customer(input: CustomerLookupInput, ctx: Context) -> CustomerInfo:
    customer = CUSTOMERS.get(input.customer_id)
    if customer is None:
        return CustomerInfo(
            customer_id=input.customer_id,
            name="Unknown",
            email="unknown@example.com",
            plan="none",
            account_status="not_found",
            default_order_id="",
            support_tier="standard",
        )
    return customer


# !!


# > Check order status
@hatchet.task(
    name="check-order-status",
    input_validator=OrderStatusInput,
    description="Check the current status, carrier, and any known issues for an order.",
)
async def check_order_status(input: OrderStatusInput, ctx: Context) -> OrderStatus:
    order = ORDERS.get(input.order_id)
    if order is None:
        return OrderStatus(
            order_id=input.order_id,
            status="not_found",
            last_updated="",
            estimated_delivery="",
            known_issue=None,
            carrier="unknown",
            tracking_number="",
        )
    return order


# !!


# > Create ticket
@hatchet.task(
    name="create-ticket",
    input_validator=CreateTicketInput,
    description="Create a support ticket for a customer issue and return the ticket ID and routing.",
)
async def create_ticket(input: CreateTicketInput, ctx: Context) -> TicketResult:
    ticket_id = f"TICKET-{input.customer_id}-001"
    return TicketResult(
        ticket_id=ticket_id,
        status="open",
        priority=input.priority,
        routing_team="shipping-support",
        summary=f"Ticket {ticket_id} created for {input.customer_id} "
        f"regarding order {input.order_id}: {input.subject}",
    )


# !!


# > Create Claude tools
def create_lookup_customer_tool_claude() -> SdkMcpTool[CustomerLookupInput]:
    return lookup_customer.mcp_tool(MCPProvider.CLAUDE)


def create_check_order_status_tool_claude() -> SdkMcpTool[OrderStatusInput]:
    return check_order_status.mcp_tool(MCPProvider.CLAUDE)


def create_ticket_tool_claude() -> SdkMcpTool[CreateTicketInput]:
    return create_ticket.mcp_tool(MCPProvider.CLAUDE)


# !!


# > Create openai tools
def create_lookup_customer_tool_openai() -> FunctionTool:
    return lookup_customer.mcp_tool(MCPProvider.OPENAI)


def create_check_order_status_tool_openai() -> FunctionTool:
    return check_order_status.mcp_tool(MCPProvider.OPENAI)


def create_ticket_tool_openai() -> FunctionTool:
    return create_ticket.mcp_tool(MCPProvider.OPENAI)


# !!
