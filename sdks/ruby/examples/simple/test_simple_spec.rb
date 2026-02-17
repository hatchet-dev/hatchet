# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "SimpleWorkflow" do
  it "runs simple task via run" do
    result = SIMPLE.run
    expect(result).to eq({ "result" => "Hello, world!" })
  end

  it "runs simple task via run_no_wait" do
    ref = SIMPLE.run_no_wait
    result = ref.result
    expect(result).to eq({ "result" => "Hello, world!" })
  end

  it "runs simple task via run_many" do
    results = SIMPLE.run_many([SIMPLE.create_bulk_run_item])
    expect(results.first).to eq({ "result" => "Hello, world!" })
  end

  it "runs simple durable task" do
    result = SIMPLE_DURABLE.run
    expect(result).to eq({ "result" => "Hello, world!" })
  end
end
