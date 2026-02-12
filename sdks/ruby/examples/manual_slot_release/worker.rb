# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new

# > SlotRelease
SLOT_RELEASE_WORKFLOW = HATCHET.workflow(name: "SlotReleaseWorkflow")

SLOT_RELEASE_WORKFLOW.task(:step1) do |input, ctx|
  puts "RESOURCE INTENSIVE PROCESS"
  sleep 10

  # Release the slot after the resource-intensive process, so that other steps can run
  ctx.release_slot

  puts "NON RESOURCE INTENSIVE PROCESS"
  { "status" => "success" }
end

# !!
