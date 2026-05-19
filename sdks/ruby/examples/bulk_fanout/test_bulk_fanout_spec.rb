# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "BulkFanoutParent" do
  it "bulk fans out to child workflows" do
    result = BULK_PARENT_WF.run({ "n" => 10 })

    results = result["spawn"]["results"]
    expect(results.length).to eq(10)
  end
end
