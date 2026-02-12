# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new

# Unit test workflow definitions
SYNC_STANDALONE = hatchet.task(name: "sync_standalone") do |input, ctx|
  {
    "key" => input["key"],
    "number" => input["number"],
    "additional_metadata" => ctx.additional_metadata,
    "retry_count" => ctx.retry_count,
    "mock_db_url" => ctx.lifespan&.dig("mock_db_url")
  }
end

ASYNC_STANDALONE = hatchet.task(name: "async_standalone") do |input, ctx|
  {
    "key" => input["key"],
    "number" => input["number"],
    "additional_metadata" => ctx.additional_metadata,
    "retry_count" => ctx.retry_count,
    "mock_db_url" => ctx.lifespan&.dig("mock_db_url")
  }
end

DURABLE_SYNC_STANDALONE = hatchet.durable_task(name: "durable_sync_standalone") do |input, ctx|
  {
    "key" => input["key"],
    "number" => input["number"],
    "additional_metadata" => ctx.additional_metadata,
    "retry_count" => ctx.retry_count,
    "mock_db_url" => ctx.lifespan&.dig("mock_db_url")
  }
end

DURABLE_ASYNC_STANDALONE = hatchet.durable_task(name: "durable_async_standalone") do |input, ctx|
  {
    "key" => input["key"],
    "number" => input["number"],
    "additional_metadata" => ctx.additional_metadata,
    "retry_count" => ctx.retry_count,
    "mock_db_url" => ctx.lifespan&.dig("mock_db_url")
  }
end

SIMPLE_UNIT_TEST_WORKFLOW = hatchet.workflow(name: "simple-unit-test-workflow")

SIMPLE_UNIT_TEST_WORKFLOW.task(:sync_simple_workflow) do |input, ctx|
  {
    "key" => input["key"],
    "number" => input["number"],
    "additional_metadata" => ctx.additional_metadata,
    "retry_count" => ctx.retry_count,
    "mock_db_url" => ctx.lifespan&.dig("mock_db_url")
  }
end

COMPLEX_UNIT_TEST_WORKFLOW = hatchet.workflow(name: "complex-unit-test-workflow")

UNIT_START = COMPLEX_UNIT_TEST_WORKFLOW.task(:start) do |input, ctx|
  {
    "key" => input["key"],
    "number" => input["number"],
    "additional_metadata" => ctx.additional_metadata,
    "retry_count" => ctx.retry_count,
    "mock_db_url" => ctx.lifespan&.dig("mock_db_url")
  }
end

COMPLEX_UNIT_TEST_WORKFLOW.task(:sync_complex_workflow, parents: [UNIT_START]) do |input, ctx|
  ctx.task_output(UNIT_START)
end
