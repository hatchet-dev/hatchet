# frozen_string_literal: true

require "securerandom"
require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "ConcurrencyWorkflowManyKeys" do
  CHARACTERS = %w[Anna Vronsky Stiva Dolly Levin Karenin].freeze
  DIGITS = (0..5).map(&:to_s).freeze

  def are_overlapping?(x, y)
    (x[:started_at] < y[:finished_at] && x[:finished_at] > y[:started_at]) ||
      (x[:finished_at] > y[:started_at] && x[:started_at] < y[:started_at])
  end

  def valid_group?(group)
    digits = Hash.new(0)
    names = Hash.new(0)

    group.each do |task|
      digits[task[:digit]] += 1
      names[task[:name]] += 1
    end

    return false if digits.values.any? { |v| v > DIGIT_MAX_RUNS }
    return false if names.values.any? { |v| v > NAME_MAX_RUNS }

    true
  end

  it "respects multiple concurrency keys" do
    test_run_id = SecureRandom.uuid

    run_refs = CONCURRENCY_MULTIPLE_KEYS_WORKFLOW.run_many_no_wait(
      100.times.map do
        name = CHARACTERS.sample
        digit = DIGITS.sample

        CONCURRENCY_MULTIPLE_KEYS_WORKFLOW.create_bulk_run_item(
          input: { "name" => name, "digit" => digit },
          options: Hatchet::TriggerWorkflowOptions.new(
            additional_metadata: {
              "test_run_id" => test_run_id,
              "key" => "#{name}-#{digit}",
              "name" => name,
              "digit" => digit
            }
          )
        )
      end
    )

    run_refs.each(&:result)

    workflows = hatchet.workflows.list(
      workflow_name: CONCURRENCY_MULTIPLE_KEYS_WORKFLOW.name,
      limit: 1000
    ).rows

    expect(workflows).not_to be_empty

    workflow = workflows.find { |w| w.name == CONCURRENCY_MULTIPLE_KEYS_WORKFLOW.name }
    expect(workflow).not_to be_nil

    runs = hatchet.runs.list(
      workflow_ids: [workflow.metadata.id],
      additional_metadata: { "test_run_id" => test_run_id },
      limit: 1000
    )

    sorted_runs = runs.rows.map do |r|
      {
        key: (r.additional_metadata || {})["key"],
        name: (r.additional_metadata || {})["name"],
        digit: (r.additional_metadata || {})["digit"],
        started_at: r.started_at,
        finished_at: r.finished_at
      }
    end.sort_by { |r| r[:started_at] }

    overlapping_groups = {}

    sorted_runs.each do |run|
      has_group_membership = false

      if overlapping_groups.empty?
        overlapping_groups[1] = [run]
        next
      end

      overlapping_groups.each do |id, group|
        if group.all? { |task| are_overlapping?(run, task) }
          overlapping_groups[id] << run
          has_group_membership = true
          break
        end
      end

      unless has_group_membership
        overlapping_groups[overlapping_groups.size + 1] = [run]
      end
    end

    overlapping_groups.each do |id, group|
      expect(valid_group?(group)).to be(true), "Group #{id} is not valid"
    end
  end
end
