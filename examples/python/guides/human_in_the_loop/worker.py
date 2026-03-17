from hatchet_sdk import DurableContext, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

APPROVAL_EVENT_KEY = "approval:decision"


# > Step 02 Wait For Event
async def wait_for_approval(ctx: DurableContext) -> dict:
    run_id = ctx.workflow_run_id
    approval = await ctx.aio_wait_for_event(
        APPROVAL_EVENT_KEY,
        f"input.runId == '{run_id}'",
    )
    return approval




# > Step 01 Define Approval Task
@hatchet.durable_task(name="ApprovalTask")
async def approval_task(input: EmptyModel, ctx: DurableContext) -> dict:
    proposed_action = {"action": "send_email", "to": "user@example.com"}
    approval = await wait_for_approval(ctx)
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
