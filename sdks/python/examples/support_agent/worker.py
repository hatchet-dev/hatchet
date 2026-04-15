import os
from datetime import timedelta
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import (
    Context,
    DurableContext,
    Hatchet,
    SleepCondition,
    UserEventCondition,
    or_,
)

hatchet = Hatchet()

REPLY_EVENT_KEY = "support:customer-reply"
TIMEOUT_SECONDS = 5


# > Models
class SupportTicketInput(BaseModel):
    ticket_id: str
    customer_email: str
    subject: str
    body: str


class TriageOutput(BaseModel):
    category: str
    priority: str


class ReplyOutput(BaseModel):
    message: str


class EscalationOutput(BaseModel):
    reason: str
    assigned_to: str


# !!


# > Triage task
@hatchet.task(input_validator=SupportTicketInput)
async def triage_ticket(input: SupportTicketInput, ctx: Context) -> TriageOutput:
    """Classify the ticket into a category and priority."""
    subject = input.subject.lower()
    body = input.body.lower()
    text = subject + " " + body

    if any(word in text for word in ["bill", "charge", "payment", "invoice"]):
        category = "billing"
    elif any(word in text for word in ["login", "password", "auth", "access"]):
        category = "account"
    else:
        category = "technical"

    if any(word in text for word in ["urgent", "critical", "down", "outage"]):
        priority = "high"
    elif any(word in text for word in ["twice", "broken", "error"]):
        priority = "medium"
    else:
        priority = "low"

    return TriageOutput(category=category, priority=priority)


# !!


# > Generate reply task
@hatchet.task(input_validator=SupportTicketInput)
async def generate_reply(input: SupportTicketInput, ctx: Context) -> ReplyOutput:
    """Generate an initial support reply using Claude."""
    api_key = os.environ.get("ANTHROPIC_API_KEY")

    if not api_key:
        return ReplyOutput(
            message=f"Thank you for contacting support about: {input.subject}. "
            "We are looking into this and will get back to you shortly."
        )

    import importlib

    anthropic = importlib.import_module("anthropic")
    client = anthropic.AsyncAnthropic(api_key=api_key)

    response = await client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=300,
        messages=[
            {
                "role": "user",
                "content": (
                    f"You are a friendly support agent. Write a brief, helpful initial "
                    f"reply to this support ticket.\n\n"
                    f"Subject: {input.subject}\n"
                    f"Message: {input.body}\n\n"
                    f"Keep the reply under 3 sentences."
                ),
            }
        ],
    )

    text = response.content[0].text
    return ReplyOutput(message=text)


# !!


# > Escalate task
@hatchet.task(input_validator=SupportTicketInput)
async def escalate_ticket(input: SupportTicketInput, ctx: Context) -> EscalationOutput:
    """Escalate an unresolved ticket to the human support team."""
    return EscalationOutput(
        reason=f"No customer reply within {TIMEOUT_SECONDS}s timeout",
        assigned_to="support-team@example.com",
    )


# !!


# > Support agent workflow
@hatchet.durable_task(input_validator=SupportTicketInput)
async def support_agent(
    input: SupportTicketInput, ctx: DurableContext
) -> dict[str, Any]:
    # Step 1: Triage the ticket
    triage = await triage_ticket.aio_run(input)

    # Step 2: Generate an initial reply
    reply = await generate_reply.aio_run(input)

    # Step 3: Wait for a customer reply or timeout
    now = await ctx.aio_now()
    consider_events_since = now - timedelta(minutes=5)

    wait_result = await ctx.aio_wait_for(
        "await-customer-reply",
        or_(
            SleepCondition(timedelta(seconds=TIMEOUT_SECONDS)),
            UserEventCondition(
                event_key=REPLY_EVENT_KEY,
                scope=input.ticket_id,
                consider_events_since=consider_events_since,
            ),
        ),
    )

    # The or-group result is {"CREATE": {"<condition_key>": ...}}.
    # Check whether the reply event condition was the one that resolved.
    resolved_key = list(wait_result["CREATE"].keys())[0]
    customer_replied = resolved_key == REPLY_EVENT_KEY

    if not customer_replied:
        # Step 4a: Timeout -> escalate
        await escalate_ticket.aio_run(input)
        return {
            "ticket_id": input.ticket_id,
            "status": "escalated",
            "triage_category": triage.category,
            "triage_priority": triage.priority,
            "initial_reply": reply.message,
        }

    # Step 4b: Customer replied -> resolve
    return {
        "ticket_id": input.ticket_id,
        "status": "resolved",
        "triage_category": triage.category,
        "triage_priority": triage.priority,
        "initial_reply": reply.message,
    }


# !!


# > Worker registration
def main() -> None:
    worker = hatchet.worker(
        "support-agent-worker",
        workflows=[support_agent, triage_ticket, generate_reply, escalate_ticket],
    )
    worker.start()


if __name__ == "__main__":
    main()


# !!
