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

# Poll until the workflow run reaches RUNNING status or timeout is exceeded.
# Retries on 404s since the run record may not be immediately visible.
def wait_for_running_status(client, run_id, timeout: 60, interval: 0.5)
  max_iters = (timeout / interval).to_i
  max_iters.times do
    begin
      details = client.runs.get_details(run_id)
      return if details.run&.status == "RUNNING"
    rescue HatchetSdkRest::ApiError => e
      raise unless e.code == 404
    end
    sleep interval
  end
end
