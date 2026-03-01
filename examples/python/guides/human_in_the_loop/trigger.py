from hatchet_sdk import Hatchet

hatchet = Hatchet(debug=True)


# > Step 03 Push Approval Event
# Your frontend or API pushes the approval event when the human clicks Approve/Reject.
# Use the same event key the task is waiting for.
def push_approval(approved: bool, reason: str = "") -> None:
    hatchet.event.push(
        "approval:decision",
        {"approved": approved, "reason": reason},
    )


# Approve: push_approval(True)
# Reject:  push_approval(False, reason="needs review")
