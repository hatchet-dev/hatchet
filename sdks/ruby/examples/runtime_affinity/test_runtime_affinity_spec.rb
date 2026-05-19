# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "../worker_fixture"
require_relative "worker"

LABELS = %w[foo bar].freeze

RSpec.describe "RuntimeAffinity" do
  around(:all) do |example|
    HatchetWorkerFixture.with_worker(
      ["bundle", "exec", "ruby", File.expand_path("worker.rb", __dir__), "--label", LABELS[0]],
      healthcheck_port: 8003
    ) do |_pid_a|
      HatchetWorkerFixture.with_worker(
        ["bundle", "exec", "ruby", File.expand_path("worker.rb", __dir__), "--label", LABELS[1]],
        healthcheck_port: 8004
      ) do |_pid_b|
        example.run
      end
    end
  end

  it "routes runs to the correct worker based on desired labels" do
    workers_list = hatchet.workers.list
    active_workers = (workers_list.rows || []).select do |w|
      w.status == "ACTIVE" &&
        w.name == hatchet.config.apply_namespace("runtime-affinity-worker")
    end

    expect(active_workers.length).to eq(2)

    worker_label_to_id = {}
    active_workers.each do |w|
      (w.labels || []).each do |label|
        if label.key == "affinity" && LABELS.include?(label.value)
          worker_label_to_id[label.value] = w.metadata.id
        end
      end
    end

    expect(worker_label_to_id.keys.sort).to eq(LABELS.sort)

    20.times do
      target_worker = LABELS.sample

      result = AFFINITY_EXAMPLE_TASK.run(
        {},
        options: Hatchet::TriggerWorkflowOptions.new(
          desired_worker_labels: {
            "affinity" => Hatchet::DesiredWorkerLabel.new(
              value: target_worker,
              required: true
            )
          }
        )
      )

      expect(result["worker_id"]).to eq(worker_label_to_id[target_worker])
    end
  end
end
