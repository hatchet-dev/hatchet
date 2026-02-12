# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "LifespanWorkflow" do
  it "injects lifespan context into tasks" do
    result = LIFESPAN_TASK.run

    expect(result["foo"]).to eq("bar")
    expect(result["pi"]).to eq(3.14)
  end
end
