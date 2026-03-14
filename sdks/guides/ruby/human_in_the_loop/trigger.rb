# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 03 Push Approval Event
# Include the run_id so the event matches the specific task waiting for it.
def push_approval(run_id:, approved:, reason: "")
  HATCHET.events.create(
    key: "approval:decision",
    data: { "runId" => run_id, "approved" => approved, "reason" => reason },
  )
end

# Approve: push_approval(run_id: 'run-id-from-ui', approved: true)
# Reject:  push_approval(run_id: 'run-id-from-ui', approved: false, reason: "needs review")
# !!
