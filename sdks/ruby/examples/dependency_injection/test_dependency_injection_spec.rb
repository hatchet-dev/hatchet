# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "DependencyInjection" do
  let(:expected_output) do
    {
      "sync_dep" => SYNC_DEPENDENCY_VALUE,
      "async_dep" => ASYNC_DEPENDENCY_VALUE,
      "async_cm_dep" => "#{ASYNC_CM_DEPENDENCY_VALUE}_#{ASYNC_DEPENDENCY_VALUE}",
      "sync_cm_dep" => "#{SYNC_CM_DEPENDENCY_VALUE}_#{SYNC_DEPENDENCY_VALUE}",
      "chained_dep" => "chained_#{CHAINED_CM_VALUE}",
      "chained_async_dep" => "chained_#{CHAINED_ASYNC_CM_VALUE}"
    }
  end

  [
    ["async_task_with_dependencies", :ASYNC_TASK_WITH_DEPS],
    ["sync_task_with_dependencies", :SYNC_TASK_WITH_DEPS],
    ["durable_async_task_with_dependencies", :DURABLE_ASYNC_TASK_WITH_DEPS],
    ["durable_sync_task_with_dependencies", :DURABLE_SYNC_TASK_WITH_DEPS]
  ].each do |name, const|
    it "resolves dependencies for #{name}" do
      task = Object.const_get(const)
      result = task.run

      expect(result).to eq(expected_output)
    end
  end
end
