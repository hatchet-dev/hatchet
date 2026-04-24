# frozen_string_literal: true

require "spec_helper"
require "timeout"

RSpec.describe Hatchet::WorkerRuntime::Runner do
  let(:dispatcher_client) { double("dispatcher_client") }
  let(:event_client) { double("event_client") }
  let(:logger) { instance_double(Logger, debug: nil, info: nil, warn: nil, error: nil) }
  let(:client) { double("client") }

  subject(:runner) do
    described_class.new(
      workflows: [],
      slots: 1,
      dispatcher_client: dispatcher_client,
      event_client: event_client,
      logger: logger,
      client: client,
    )
  end

  after do
    runner.send(:stop_step_action_event_thread)
  end

  it "sends STARTED asynchronously and includes retry_count" do
    action = double("action", retry_count: 3)
    started = Queue.new

    allow(dispatcher_client).to receive(:send_step_action_event) do |**kwargs|
      sleep 0.1
      started << kwargs
    end

    before = Process.clock_gettime(Process::CLOCK_MONOTONIC)
    runner.send(:send_started, action)
    elapsed = Process.clock_gettime(Process::CLOCK_MONOTONIC) - before

    expect(elapsed).to be < 0.05

    payload = Timeout.timeout(1) { started.pop }
    expect(payload).to include(
      action: action,
      event_type: :STEP_EVENT_TYPE_STARTED,
      payload: "{}",
      retry_count: 3,
    )
  end

  it "retries STARTED delivery on transient errors" do
    action = double("action", retry_count: 2)
    attempts = Queue.new
    call_count = 0

    allow(runner).to receive(:started_event_backoff_seconds).and_return(0)
    allow(dispatcher_client).to receive(:send_step_action_event) do |**kwargs|
      call_count += 1
      attempts << kwargs
      raise StandardError, "temporary" if call_count == 1
    end

    runner.send(:send_started, action)

    first = Timeout.timeout(1) { attempts.pop }
    second = Timeout.timeout(1) { attempts.pop }

    expect(first).to include(retry_count: 2)
    expect(second).to include(retry_count: 2)
    expect(dispatcher_client).to have_received(:send_step_action_event).twice
  end

  it "initializes the eviction manager only once under concurrent startup" do
    config = double("config")
    channel = double("channel")
    allow(config).to receive(:apply_namespace).and_return("test")
    allow(client).to receive(:config).and_return(config)
    allow(client).to receive(:channel).and_return(channel)

    workflow = Hatchet::Workflow.new(name: "TestWorkflow", client: client)
    workflow.durable_task(name: "durable_task", execution_timeout: 60) { |_input, _ctx| { "ok" => true } }

    durable_runner = described_class.new(
      workflows: [workflow],
      slots: 2,
      dispatcher_client: dispatcher_client,
      event_client: event_client,
      logger: logger,
      client: client,
      engine_version: Hatchet::MinEngineVersion::DURABLE_EVICTION,
    )

    managers = Queue.new
    allow(Hatchet::WorkerRuntime::DurableEviction::DurableEvictionManager).to receive(:new) do |**_kwargs|
      sleep 0.05
      instance_double(Hatchet::WorkerRuntime::DurableEviction::DurableEvictionManager, start: nil).tap do |manager|
        managers << manager
      end
    end

    ready = Queue.new
    release = Queue.new
    threads = 2.times.map do
      Thread.new do
        ready << true
        release.pop
        durable_runner.send(:ensure_eviction_manager_started, nil)
      end
    end

    2.times { ready.pop }
    2.times { release << true }
    threads.each(&:join)

    created_manager = managers.pop

    expect(Hatchet::WorkerRuntime::DurableEviction::DurableEvictionManager).to have_received(:new).once
    expect(durable_runner.eviction_manager).to be(created_manager)
  ensure
    durable_runner&.send(:stop_step_action_event_thread)
  end
end
