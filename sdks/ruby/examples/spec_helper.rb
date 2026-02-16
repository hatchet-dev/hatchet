# frozen_string_literal: true

require "hatchet-sdk"
require_relative "worker_fixture"

RSpec.configure do |config|
  config.expect_with :rspec do |expectations|
    expectations.include_chain_clauses_in_custom_matcher_descriptions = true
  end

  config.mock_with :rspec do |mocks|
    mocks.verify_partial_doubles = true
  end

  config.shared_context_metadata_behavior = :apply_to_host_groups
  config.filter_run_when_matching :focus
  config.order = :random

  # Session-scoped Hatchet client
  config.add_setting :hatchet_client
  config.before(:suite) do
    RSpec.configuration.hatchet_client = Hatchet::Client.new(debug: true)
  end
end

# Helper to access the shared Hatchet client in tests
def hatchet
  RSpec.configuration.hatchet_client
end
