from uuid import uuid4

import pytest

from examples.support_agent.worker import (
    REPLY_EVENT_KEY,
    SupportTicketInput,
    support_agent,
)
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_support_agent_reply_before_timeout(hatchet: Hatchet) -> None:
    """Customer replies before timeout -> workflow resolves."""
    ticket_id = f"test-{uuid4().hex[:8]}"
    ticket = SupportTicketInput(
        ticket_id=ticket_id,
        customer_email="alice@example.com",
        subject="Login broken",
        body="I can't log in since this morning.",
    )

    ref = await support_agent.aio_run(ticket, wait_for_result=False)

    # The workflow uses consider_events_since lookback, so the reply event
    # is captured even if pushed before the or-group wait becomes active.
    await hatchet.event.aio_push(
        REPLY_EVENT_KEY,
        {"message": "Actually, I just needed to clear my cookies. Fixed!"},
        scope=ticket.ticket_id,
    )

    result = await ref.aio_result()

    assert result["ticket_id"] == ticket_id
    assert result["status"] == "resolved"
    assert result["triage_category"] is not None
    assert result["initial_reply"] is not None


@pytest.mark.asyncio(loop_scope="session")
async def test_support_agent_timeout_escalation(hatchet: Hatchet) -> None:
    """No customer reply -> workflow escalates after timeout."""
    ticket_id = f"test-{uuid4().hex[:8]}"
    ticket = SupportTicketInput(
        ticket_id=ticket_id,
        customer_email="bob@example.com",
        subject="Billing issue",
        body="I was charged twice for my subscription.",
    )

    result = await support_agent.aio_run(ticket)

    assert result["ticket_id"] == ticket_id
    assert result["status"] == "escalated"
    assert result["triage_category"] is not None
    assert result["initial_reply"] is not None
