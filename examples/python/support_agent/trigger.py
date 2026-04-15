# > Trigger the workflow
from examples.support_agent.worker import (
    REPLY_EVENT_KEY,
    SupportTicketInput,
    hatchet,
    support_agent,
)

ticket = SupportTicketInput(
    ticket_id="ticket-42",
    customer_email="alice@example.com",
    subject="Login broken",
    body="I can't log in since this morning.",
)

# Start the support agent workflow
ref = support_agent.run(ticket, wait_for_result=False)
print(f"Started workflow run: {ref.workflow_run_id}")

# Push a customer reply event (scoped to this ticket)
print("Pushing customer reply event...")
hatchet.event.push(
    REPLY_EVENT_KEY,
    {"message": "I cleared my cookies and it works now. Thanks!"},
    scope=ticket.ticket_id,
)

# Wait for the workflow to complete
result = ref.result()
print(f"Workflow completed: {result}")


