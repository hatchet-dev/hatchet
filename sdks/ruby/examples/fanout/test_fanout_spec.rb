# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "FanoutParent" do
  it "fans out to child workflows" do
    result = FANOUT_PARENT_WF.run({ "n" => 5 })

    results = result["spawn"]["results"]
    expect(results.length).to eq(5)
  end
end
