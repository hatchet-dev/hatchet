# frozen_string_literal: true

require "hatchet-sdk"
require_relative "worker"

hatchet = Hatchet::Client.new

# > Running a task from within a task
SPAWN_TASK = hatchet.task(name: "SpawnTask") do |input, ctx|
  result = CHILD_TASK_WF.run({ "message" => "Hello, World!" })
  { "results" => result }
end
# !!
