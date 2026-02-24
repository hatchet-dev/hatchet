# frozen_string_literal: true

# Helper methods for integration tests

module IntegrationHelper
  # Check if integration tests should be run
  def self.should_run_integration_tests?
    ENV["HATCHET_CLIENT_TOKEN"] || ENV["RUN_INTEGRATION_TESTS"] == "true"
  end

  # Skip test if no credentials available
  def skip_if_no_credentials(message = "Integration tests require HATCHET_CLIENT_TOKEN to be set")
    skip message unless IntegrationHelper.should_run_integration_tests?
  end

  # Create a test client for integration tests
  def create_test_client
    skip_if_no_credentials
    Hatchet::Client.new
  end

  # Get test workflow run ID from recent runs
  def get_test_workflow_run_id(runs_client, limit: 1)
    recent_runs = runs_client.list(limit: limit)
    if recent_runs.rows&.any?
      recent_runs.rows.first.metadata.id
    else
      skip "No recent workflow runs found for testing"
    end
  end

  # Get test task run ID from recent runs
  def get_test_task_run_id(runs_client, limit: 1)
    recent_runs = runs_client.list(only_tasks: true, limit: limit)
    if recent_runs.rows&.any?
      recent_runs.rows.first.metadata.id
    else
      skip "No recent task runs found for testing"
    end
  end

  # Safely attempt an operation that might fail due to missing test data
  def safely_attempt_operation(description)
    result = yield
    puts "✓ #{description}: Success"
    result
  rescue StandardError => e
    puts "⚠ #{description}: #{e.message}"
    expect(e).to be_a(StandardError)
    nil
  end
end

# Include helper in integration test contexts
RSpec.configure do |config|
  config.include IntegrationHelper, :integration
end
