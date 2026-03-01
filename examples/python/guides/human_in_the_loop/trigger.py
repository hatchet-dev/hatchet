from hatchet_sdk import Hatchet

hatchet = Hatchet(debug=True)


# > Step 03 Push Approval Event
# Include the run_id so the event matches the specific task waiting for it.
def push_approval(run_id: str, approved: bool, reason: str = "") -> None:
    hatchet.event.push(
        "approval:decision",
        {"runId": run_id, "approved": approved, "reason": reason},
    )


# Approve: push_approval("run-id-from-ui", True)
# Reject:  push_approval("run-id-from-ui", False, reason="needs review")
