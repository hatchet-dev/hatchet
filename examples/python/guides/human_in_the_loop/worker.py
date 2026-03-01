from hatchet_sdk import DurableContext, EmptyModel, Hatchet, UserEventCondition

hatchet = Hatchet(debug=True)

APPROVAL_EVENT_KEY = "approval:decision"


# > Step 01 Define Approval Task
@hatchet.durable_task(name="ApprovalTask")
async def approval_task(input: EmptyModel, ctx: DurableContext) -> dict:
    """Propose an action and wait for human approval."""
    proposed_action = {"action": "send_email", "to": "user@example.com"}
    # Task will pause at wait_for until event arrives
    approval = await ctx.aio_wait_for(
        "approval",
        UserEventCondition(event_key=APPROVAL_EVENT_KEY),
    )
    if approval.get("approved"):
        return {"status": "approved", "action": proposed_action}
    return {"status": "rejected", "reason": approval.get("reason", "")}




# > Step 02 Wait For Event
# Pause until the approval event is pushed. Worker slot is freed while waiting.
async def _wait_for_approval(ctx: DurableContext, proposed_action: dict) -> dict:
    approval = await ctx.aio_wait_for("approval", UserEventCondition(event_key=APPROVAL_EVENT_KEY))
    if approval.get("approved"):
        return {"status": "approved", "action": proposed_action}
    return {"status": "rejected", "reason": approval.get("reason", "")}


def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "human-in-the-loop-worker",
        workflows=[approval_task],
    )
    worker.start()


if __name__ == "__main__":
    main()
