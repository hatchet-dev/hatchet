# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "../worker_fixture"
require_relative "worker"

RSpec.describe "BatchAssign" do
  around do |example|
    HatchetWorkerFixture.with_worker(
      ["bundle", "exec", "ruby", File.expand_path("worker.rb", __dir__)]
    ) do
      example.run
    end
  end

  # Run blocks concurrently via Thread.new (Task#run blocks until the result is ready),
  # mirroring the concurrent fan-out used by the Python reference tests (asyncio.gather).
  def concurrently(items)
    threads = items.map.with_index { |item, i| Thread.new { yield(item, i) } }
    threads.map(&:value)
  end

  it "flushes when batch size is reached" do
    inputs = %w[alpha bravo charlie]

    results = concurrently(inputs) { |msg, _i| BATCH_SIMPLE.run({ "message" => msg }) }

    expect(results.map { |r| r["transformed_message"] }).to eq(inputs.map(&:upcase))
  end

  it "flushes on interval when fewer items than batch size are buffered" do
    inputs = %w[delta echo]

    refs = inputs.map { |msg| BATCH_SIMPLE.run_no_wait({ "message" => msg }) }
    sleep 0.5

    results = refs.map(&:result)

    expect(results.map { |r| r["transformed_message"] }).to eq(inputs.map(&:upcase))
  end

  it "partitions batches by key when batch size is reached" do
    inputs = [
      { "message" => "alpha", "group" => "tenant-1" },
      { "message" => "bravo", "group" => "tenant-1" },
      { "message" => "charlie", "group" => "tenant-2" },
      { "message" => "delta", "group" => "tenant-2" },
    ]

    results = concurrently(inputs) { |input, _i| BATCH_KEYED.run(input) }

    inputs.each_with_index do |input, i|
      expect(results[i]["batch_key"]).to eq(input["group"])
      expect(results[i]["batch_size"]).to eq(2)
      expect(results[i]["unique_keys"]).to eq(1)
      expect(results[i]["uppercase"]).to eq(input["message"].upcase)
    end
  end

  it "fails only the task whose batch group key fails to parse" do
    good_ref = BATCH_KEYED_FAILABLE.run_no_wait({ "message" => "hello", "group" => "tenant-1" })
    bad_ref = BATCH_KEYED_FAILABLE.run_no_wait({ "message" => "world", "group" => 123 })

    expect { bad_ref.result }.to raise_error(/failed to parse batch group key expression/)

    good_result = good_ref.result
    expect(good_result["uppercase"]).to eq("HELLO")
  end

  it "flushes keyed batches independently when interval elapses" do
    inputs = [
      { "message" => "echo", "group" => "tenant-1" },
      { "message" => "foxtrot", "group" => "tenant-1" },
      { "message" => "golf", "group" => "tenant-1" },
      { "message" => "hotel", "group" => "tenant-2" },
    ]

    results = concurrently(inputs) { |input, _i| BATCH_KEYED_INTERVAL.run(input) }

    inputs.each_with_index { |input, i| expect(results[i]["batch_key"]).to eq(input["group"]) }
    (0..2).each { |i| expect(results[i]["batch_size"]).to eq(3) }
    expect(results[3]["batch_size"]).to eq(1)
    results.each { |r| expect(r["unique_keys"]).to eq(1) }
    expect(results[3]["uppercase"]).to eq("HOTEL")
  end

  it "completes all tasks with large payloads, flushing on memory size" do
    payload_size = 100_000
    payload = "x" * payload_size
    task_count = 100

    results = concurrently(Array.new(task_count)) { |_item, _i| BATCH_LARGE.run({ "data" => payload }) }

    expect(results.length).to eq(task_count)
    # The batch should flush 3 times due to the ~4mb memory-size limit, even though
    # batch_max_size (100) is never reached by count alone.
    expect(results.map { |r| r["batch_id"] }.uniq.length).to eq(3)
    expect(results).to all(include("received" => true))
    results.each { |r| expect(r["data_length"]).to eq(payload_size) }
  end

  it "handles batch size of one without keys" do
    inputs = %w[india juliet]

    results = concurrently(inputs) { |msg, _i| BATCH_SINGLE.run({ "message" => msg }) }

    expect(results.map { |r| r["batch_size"] }).to eq([1, 1])
    expect(results.map { |r| r["original"] }).to eq(inputs)
  end

  it "returns results in submission order" do
    count = 20

    results = concurrently((0...count).to_a) { |index, _i| BATCH_ORDERED.run({ "index" => index }) }

    results.each_with_index { |result, i| expect(result["index"]).to eq(i) }
  end

  it "broadcasts the same result to all callers" do
    count = 10

    results = concurrently(Array.new(count)) { |_item, _i| BATCH_BROADCAST.run({ "message" => "hello" }) }

    expect(results.map { |r| r["sum"] }).to all(eq(50))
  end

  it "supports in-batch cancellation via context.cancel" do
    count = 10

    results = concurrently(Array.new(count)) { |_item, _i| BATCH_CANCEL.run({ "message" => "hello" }) }

    expect(results.length).to eq(count)
  end

  it "supports spawning child tasks from a batch handler" do
    count = 10

    results = concurrently(Array.new(count)) { |_item, _i| BATCH_CHILD_SPAWN.run({ "message" => "hello" }) }

    results.each { |r| expect(r).not_to be_empty }
  end

  it "supports spawning child batch tasks from a batch handler" do
    count = 10

    results = concurrently(Array.new(count)) { |_item, _i| BATCH_CHILD_BATCH_SPAWN.run({ "message" => "hello" }) }

    results.each { |r| expect(r).not_to be_empty }
  end
end
