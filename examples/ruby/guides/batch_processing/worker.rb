# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Define Parent Task
BATCH_PARENT_WF = HATCHET.workflow(name: "BatchParent")
BATCH_CHILD_WF = HATCHET.workflow(name: "BatchChild")

BATCH_PARENT_WF.durable_task(:spawn_children) do |input, _ctx|
  items = input["items"] || []
  results = BATCH_CHILD_WF.run_many(
    items.map { |item_id| BATCH_CHILD_WF.create_bulk_run_item(input: { "item_id" => item_id }) },
  )
  { "processed" => results.size, "results" => results }
end


# > Step 03 Process Item
BATCH_CHILD_WF.task(:process_item) do |input, _ctx|
  { "status" => "done", "item_id" => input["item_id"] }
end


def main
  # > Step 04 Run Worker
  worker = HATCHET.worker("batch-worker", slots: 20, workflows: [BATCH_PARENT_WF, BATCH_CHILD_WF])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
