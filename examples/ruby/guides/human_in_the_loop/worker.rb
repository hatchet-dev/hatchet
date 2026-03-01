# frozen_string_literal: true

require 'hatchet-sdk'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

APPROVAL_EVENT_KEY = 'approval:decision'

# > Step 01 Define Approval Task
APPROVAL_TASK = HATCHET.durable_task(name: 'ApprovalTask') do |_input, ctx|
  proposed_action = { 'action' => 'send_email', 'to' => 'user@example.com' }
  approval = ctx.wait_for(
    'approval',
    Hatchet::UserEventCondition.new(event_key: APPROVAL_EVENT_KEY)
  )
  if approval['approved']
    { 'status' => 'approved', 'action' => proposed_action }
  else
    { 'status' => 'rejected', 'reason' => approval['reason'].to_s }
  end
end


# > Step 02 Wait For Event
# Pause until the approval event is pushed. Worker slot is freed while waiting.
def wait_for_approval(ctx, proposed_action)
  approval = ctx.wait_for('approval', Hatchet::UserEventCondition.new(event_key: APPROVAL_EVENT_KEY))
  if approval['approved']
    { 'status' => 'approved',
      'action' => proposed_action }
  else
    { 'status' => 'rejected',
      'reason' => approval['reason'].to_s }
  end
end

def main
  # > Step 04 Run Worker
  worker = HATCHET.worker(
    'human-in-the-loop-worker',
    workflows: [APPROVAL_TASK]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
