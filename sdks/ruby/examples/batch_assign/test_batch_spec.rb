# frozen_string_literal: true

require "securerandom"
require_relative "../spec_helper"
require_relative "../worker_fixture"

RSpec.describe "batch-task e2e" do
  BATCH_HEALTHCHECK_PORT = 8010

  before(:all) do
    @batch_run_id = SecureRandom.uuid

    # Set env vars before spawning the subprocess so both processes share the
    # same workflow names and healthcheck config.
    ENV["BATCH_RUN_ID"] = @batch_run_id
    ENV["HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT"] = BATCH_HEALTHCHECK_PORT.to_s
    ENV["HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED"] = "true"

    # Define workflow constants in this test process using the same run_id.
    require_relative "worker"

    # Start the worker subprocess (inherits env vars set above).
    worker_rb = File.expand_path("worker.rb", __dir__)
    @worker_pid = Process.spawn("bundle", "exec", "ruby", worker_rb, pgroup: true)

    HatchetWorkerFixture.wait_for_worker_health(port: BATCH_HEALTHCHECK_PORT, max_attempts: 30)
  end

  after(:all) do
    if @worker_pid
      begin
        Process.kill("TERM", -Process.getpgid(@worker_pid))
      rescue Errno::ESRCH, Errno::EPERM
        nil
      end
      begin
        Process.wait(@worker_pid)
      rescue Errno::ECHILD
        nil
      end
    end
    ENV.delete("BATCH_RUN_ID")
    ENV.delete("HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT")
    ENV.delete("HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED")
  end

  it "flushes when batch size is reached" do
    inputs = ["alpha", "bravo", "charlie"]

    threads = inputs.map { |msg| Thread.new { BATCH_SIMPLE_WF.run({ "Message" => msg }) } }
    results = threads.map(&:value)

    expect(results).to have_attributes(length: 3)
    expect(results.map { |r| r["TransformedMessage"] }).to eq(inputs.map(&:upcase))
  end

  it "flushes when fewer items are buffered than the batch size" do
    inputs = ["delta", "echo"]

    threads = inputs.map { |msg| Thread.new { BATCH_SIMPLE_WF.run({ "Message" => msg }) } }
    sleep 0.5
    results = threads.map(&:value)

    expect(results.map { |r| r["TransformedMessage"] }).to eq(inputs.map(&:upcase))
  end

  it "partitions batches by key when batch size is reached" do
    inputs = [
      { "Message" => "alpha", "group" => "tenant-1" },
      { "Message" => "bravo", "group" => "tenant-1" },
      { "Message" => "charlie", "group" => "tenant-2" },
      { "Message" => "delta", "group" => "tenant-2" },
    ]

    threads = inputs.map { |input| Thread.new { BATCH_KEYED_WF.run(input) } }
    results = threads.map(&:value)

    expect(results).to have_attributes(length: inputs.length)
    results.each_with_index do |result, i|
      expect(result["batchKey"]).to eq(inputs[i]["group"])
      expect(result["batchSize"]).to eq(2)
      expect(result["uniqueKeys"]).to eq(1)
      expect(result["uppercase"]).to eq(inputs[i]["Message"].upcase)
    end
  end

  it "flushes keyed batches independently when flush interval elapses" do
    inputs = [
      { "Message" => "echo",    "group" => "tenant-1" },
      { "Message" => "foxtrot", "group" => "tenant-1" },
      { "Message" => "golf",    "group" => "tenant-1" },
      { "Message" => "hotel",   "group" => "tenant-2" },
    ]

    threads = inputs.map { |input| Thread.new { BATCH_KEYED_IV_WF.run(input) } }
    results = threads.map(&:value)

    expect(results.map { |r| r["batchKey"] }).to eq(inputs.map { |i| i["group"] })
    expect(results[0, 3].all? { |r| r["batchSize"] == 3 }).to be true
    expect(results[3]["batchSize"]).to eq(1)
    expect(results.all? { |r| r["uniqueKeys"] == 1 }).to be true
    expect(results[3]["payload"]).to eq("hotel")
  end

  it "completes all tasks when batch contains 100 items with large payloads" do
    payload = "x" * 1_000_000
    task_count = 100

    threads = task_count.times.map { Thread.new { BATCH_LARGE_WF.run({ "data" => payload }) } }
    results = threads.map(&:value)

    expect(results).to have_attributes(length: task_count)
    expect(results.all? { |r| r["received"] }).to be true
    expect(results.all? { |r| r["dataLength"] == 1_000_000 }).to be true
    expect(results.all? { |r| r["batchSize"] == task_count }).to be true
  end

  it "handles batch size of one without keys" do
    inputs = ["india", "juliet"]

    threads = inputs.map { |msg| Thread.new { BATCH_SINGLE_WF.run({ "Message" => msg }) } }
    results = threads.map(&:value)

    expect(results.map { |r| r["batchSize"] }).to eq([1, 1])
    expect(results.map { |r| r["original"] }).to eq(inputs)
  end

  it "returns results in submission order" do
    count = 20

    threads = count.times.map { |i| Thread.new { BATCH_ORDERED_WF.run({ "index" => i }) } }
    results = threads.map(&:value)

    expect(results).to have_attributes(length: count)
    results.each_with_index do |result, i|
      expect(result["index"]).to eq(i)
    end
  end
end
