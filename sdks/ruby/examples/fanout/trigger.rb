# frozen_string_literal: true

require_relative "worker"

# > Bulk run children
def run_child_workflows(n)
  FANOUT_CHILD_WF.run_many(
    n.times.map do |i|
      FANOUT_CHILD_WF.create_bulk_run_item(
        input: { "a" => i.to_s }
      )
    end
  )
end
# !!
