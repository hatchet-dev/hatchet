# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "SyncFanoutParent" do
  it "fans out synchronously to child workflows" do
    result = SYNC_FANOUT_PARENT.run({ "n" => 5 })

    results = result["spawn"]["results"]
    expect(results.length).to eq(5)
  end
end
