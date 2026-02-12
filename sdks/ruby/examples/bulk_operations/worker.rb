# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

BULK_REPLAY_TEST_1 = HATCHET.task(name: "bulk_replay_test_1") do |input, ctx|
  puts "retrying bulk replay test task #{ctx.retry_count}"
  raise "This is a test error to trigger a retry." if ctx.retry_count == 0
end

BULK_REPLAY_TEST_2 = HATCHET.task(name: "bulk_replay_test_2") do |input, ctx|
  puts "retrying bulk replay test task #{ctx.retry_count}"
  raise "This is a test error to trigger a retry." if ctx.retry_count == 0
end

BULK_REPLAY_TEST_3 = HATCHET.task(name: "bulk_replay_test_3") do |input, ctx|
  puts "retrying bulk replay test task #{ctx.retry_count}"
  raise "This is a test error to trigger a retry." if ctx.retry_count == 0
end

def main
  worker = HATCHET.worker(
    "bulk-replay-test-worker",
    workflows: [BULK_REPLAY_TEST_1, BULK_REPLAY_TEST_2, BULK_REPLAY_TEST_3]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
