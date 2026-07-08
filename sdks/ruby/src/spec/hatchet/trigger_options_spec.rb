# frozen_string_literal: true

require "spec_helper"

RSpec.describe Hatchet::TriggerWorkflowOptions do
  describe "#display_name" do
    it "exposes the display name and includes it in to_h" do
      options = described_class.new(display_name: "Acme Corp")

      expect(options.display_name).to eq("Acme Corp")
      expect(options.to_h).to include(display_name: "Acme Corp")
    end

    it "omits display_name from to_h when not provided" do
      options = described_class.new

      expect(options.display_name).to be_nil
      expect(options.to_h).not_to have_key(:display_name)
    end
  end
end
