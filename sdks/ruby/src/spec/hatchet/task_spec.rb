# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::Task do
  let(:config) { instance_double(Hatchet::Config, apply_namespace: "test-service") }

  it "marks durable tasks as durable in workflow registration protos" do
    task = described_class.new(
      name: "durable_task",
      durable: true,
    ) { |_input, _ctx| nil }

    proto = task.to_proto("test-service", config: config)

    expect(proto.is_durable).to be(true)
  end

  it "leaves non-durable tasks unset in workflow registration protos" do
    task = described_class.new(
      name: "normal_task",
      durable: false,
    ) { |_input, _ctx| nil }

    proto = task.to_proto("test-service", config: config)

    expect(proto.is_durable).to be(false)
  end

  it "serializes batch config onto the registration proto" do
    batch = Hatchet::BatchTaskConfig.new(max_size: 3, max_interval_ms: 200, group_key: "input.group")
    task = described_class.new(
      name: "batch_task",
      batch: batch,
    ) { |inputs, _ctx| inputs }

    proto = task.to_proto("test-service", config: config)

    expect(proto.batch).not_to be_nil
    expect(proto.batch.batch_max_size).to eq(3)
    expect(proto.batch.batch_max_interval_ms).to eq(200)
    expect(proto.batch.batch_group_key).to eq("input.group")
  end

  it "forces retries to 0 for batch tasks even when retries is explicitly set" do
    batch = Hatchet::BatchTaskConfig.new(max_size: 3)
    task = described_class.new(
      name: "batch_task_with_retries",
      batch: batch,
      retries: 5,
    ) { |inputs, _ctx| inputs }

    proto = task.to_proto("test-service", config: config)

    expect(proto.retries).to eq(0)
  end

  it "leaves batch unset on the proto for non-batch tasks" do
    task = described_class.new(
      name: "normal_task",
    ) { |_input, _ctx| nil }

    proto = task.to_proto("test-service", config: config)

    expect(proto.batch).to be_nil
  end

  it "registers a task with CANCEL_QUEUED_EXCEPT_NEWEST concurrency" do
    concurrency = Hatchet::ConcurrencyExpression.new(
      expression: "input.group",
      max_runs: 1,
      limit_strategy: Hatchet::ConcurrencyLimitStrategy::CANCEL_QUEUED_EXCEPT_NEWEST,
    )
    task = described_class.new(
      name: "except_newest_task",
      concurrency: concurrency,
    ) { |_input, _ctx| nil }

    proto = task.to_proto("test-service", config: config)

    expect(proto.concurrency.length).to eq(1)
    expect(proto.concurrency[0].limit_strategy).to eq(:CANCEL_QUEUED_EXCEPT_NEWEST)
  end

  it "registers a task with CANCEL_QUEUED_EXCEPT_OLDEST concurrency" do
    concurrency = Hatchet::ConcurrencyExpression.new(
      expression: "input.group",
      max_runs: 1,
      limit_strategy: Hatchet::ConcurrencyLimitStrategy::CANCEL_QUEUED_EXCEPT_OLDEST,
    )
    task = described_class.new(
      name: "except_oldest_task",
      concurrency: concurrency,
    ) { |_input, _ctx| nil }

    proto = task.to_proto("test-service", config: config)

    expect(proto.concurrency.length).to eq(1)
    expect(proto.concurrency[0].limit_strategy).to eq(:CANCEL_QUEUED_EXCEPT_OLDEST)
  end

  it "registers every condition in an or_ group under one shared or group id" do
    task = described_class.new(
      name: "waits_for_sleep_or_event",
      wait_for: [
        Hatchet.or_(
          Hatchet::SleepCondition.new(60),
          Hatchet::UserEventCondition.new(event_key: "go"),
        ),
      ],
    ) { |_input, _ctx| nil }

    conditions = task.to_proto("test-service", config: config).conditions

    expect(conditions.sleep_conditions.length).to eq(1)
    expect(conditions.user_event_conditions.length).to eq(1)

    sleep_base = conditions.sleep_conditions[0].base
    event_base = conditions.user_event_conditions[0].base

    expect(sleep_base.action).to eq(:QUEUE)
    expect(event_base.action).to eq(:QUEUE)
    expect(sleep_base.or_group_id).not_to be_empty
    expect(sleep_base.or_group_id).to eq(event_base.or_group_id)
  end

  it "registers a nested or_ group in the group that contains it" do
    task = described_class.new(
      name: "waits_for_nested_or",
      wait_for: [
        Hatchet.or_(
          Hatchet::SleepCondition.new(1),
          Hatchet.or_(
            Hatchet::SleepCondition.new(2),
            Hatchet::UserEventCondition.new(event_key: "go"),
          ),
        ),
      ],
    ) { |_input, _ctx| nil }

    conditions = task.to_proto("test-service", config: config).conditions
    bases = conditions.sleep_conditions.map(&:base) + conditions.user_event_conditions.map(&:base)

    expect(bases.length).to eq(3)
    expect(bases.map(&:or_group_id).uniq.length).to eq(1)
  end

  it "registers a parent condition as a parent override condition" do
    parent = described_class.new(name: "start") { |_input, _ctx| nil }
    task = described_class.new(
      name: "branch",
      parents: [parent],
      skip_if: [Hatchet::ParentCondition.new(parent: parent, expression: "output.random_number > 50")],
    ) { |_input, _ctx| nil }

    conditions = task.to_proto("test-service", config: config).conditions

    expect(conditions.parent_override_conditions.length).to eq(1)

    parent_condition = conditions.parent_override_conditions[0]

    expect(parent_condition.parent_readable_id).to eq("start")
    expect(parent_condition.base.action).to eq(:SKIP)
    expect(parent_condition.base.expression).to eq("output.random_number > 50")
  end
end
