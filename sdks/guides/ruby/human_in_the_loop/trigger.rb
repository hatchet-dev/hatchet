# frozen_string_literal: true

require 'hatchet-sdk'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 03 Push Approval Event
# Your frontend or API pushes the approval event when the human clicks Approve/Reject.
def push_approval(approved:, reason: '')
  HATCHET.events.create(
    key: 'approval:decision',
    data: { 'approved' => approved, 'reason' => reason }
  )
end

# Approve: push_approval(approved: true)
# Reject:  push_approval(approved: false, reason: "needs review")
# !!
