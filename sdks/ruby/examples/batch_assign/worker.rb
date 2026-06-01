# frozen_string_literal: true

require "securerandom"
require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# Unique ID per worker process so workflow names don't conflict across test runs.
BATCH_RUN_ID = ENV.fetch("BATCH_RUN_ID", SecureRandom.uuid).freeze

# > Simple batch (non-keyed): flushes when batch size (3) is reached
BATCH_SIMPLE_WF = HATCHET.batch_task(
  name: "batch-e2e-simple-#{BATCH_RUN_ID}",
  retries: 0,
  batch_max_size: 3,
  batch_max_interval: "200ms",
) do |tasks|
  tasks.map { |(input, _ctx)| { "TransformedMessage" => input["Message"].upcase } }
end

# > Keyed batch: partitions by input.group, flushes per group when size (2) reached
BATCH_KEYED_WF = HATCHET.batch_task(
  name: "batch-e2e-keyed-#{BATCH_RUN_ID}",
  retries: 0,
  batch_max_size: 2,
  batch_max_interval: "200ms",
  batch_group_key: "input.group",
) do |tasks|
  tasks.map do |(input, _ctx)|
    {
      "batchKey" => input["group"],
      "batchSize" => tasks.length,
      "uniqueKeys" => tasks.map { |(i, _)| i["group"] }.uniq.length,
      "uppercase" => input["Message"].upcase,
    }
  end
end

# > Keyed interval batch: each group flushes independently when interval elapses
BATCH_KEYED_IV_WF = HATCHET.batch_task(
  name: "batch-e2e-keyed-interval-#{BATCH_RUN_ID}",
  retries: 0,
  batch_max_size: 3,
  batch_max_interval: "150ms",
  batch_group_key: "input.group",
) do |tasks|
  tasks.map do |(input, _ctx)|
    {
      "batchKey" => input["group"],
      "batchSize" => tasks.length,
      "uniqueKeys" => tasks.map { |(i, _)| i["group"] }.uniq.length,
      "payload" => input["Message"],
    }
  end
end

# > Large payload batch: 100 items, flushes on size
BATCH_LARGE_WF = HATCHET.batch_task(
  name: "batch-e2e-large-#{BATCH_RUN_ID}",
  retries: 0,
  batch_max_size: 100,
  batch_max_interval: "1000s",
) do |tasks|
  tasks.map do |(input, _ctx)|
    {
      "received" => true,
      "batchSize" => tasks.length,
      "dataLength" => input["data"].length,
    }
  end
end

# > Single-item batch: batch size of 1 (every item is its own batch)
BATCH_SINGLE_WF = HATCHET.batch_task(
  name: "batch-e2e-single-#{BATCH_RUN_ID}",
  retries: 0,
  batch_max_size: 1,
  batch_max_interval: "100ms",
) do |tasks|
  tasks.map { |(input, _ctx)| { "original" => input["Message"], "batchSize" => tasks.length } }
end

# !!

def main
  all_workflows = [
    BATCH_SIMPLE_WF,
    BATCH_KEYED_WF,
    BATCH_KEYED_IV_WF,
    BATCH_LARGE_WF,
    BATCH_SINGLE_WF,
  ]

  # Use 100 slots so batchMaxSize=100 tests don't deadlock (all items can wait simultaneously).
  worker = HATCHET.worker("batch-e2e-worker-#{BATCH_RUN_ID}", slots: 100, workflows: all_workflows)
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
