# frozen_string_literal: true

require "rspec"
require "hatchet-sdk"

RSpec.configure do |config|
  # Enable flags like --only-failures and --next-failure
  config.example_status_persistence_file_path = ".rspec_status"

  # Disable RSpec exposing methods globally on `Module` and `main`
  config.disable_monkey_patching!

  config.expect_with :rspec do |c|
    c.syntax = :expect
  end

  # Configure integration test filtering
  # Integration tests are tagged with :integration and require real API credentials
  config.filter_run_excluding :integration unless ENV["RUN_INTEGRATION_TESTS"] == "true" || ENV["HATCHET_CLIENT_TOKEN"]
  
  # Add some helpful output for integration tests
  config.before(:suite) do
    if ENV["HATCHET_CLIENT_TOKEN"] && RSpec.configuration.inclusion_filter[:integration]
      puts "\n🔗 Running integration tests with real API credentials"
      puts ""
    elsif RSpec.configuration.exclusion_filter[:integration]
      puts "\n⚠️  Integration tests skipped (no HATCHET_CLIENT_TOKEN found)"
      puts "   Set HATCHET_CLIENT_TOKEN or RUN_INTEGRATION_TESTS=true to run integration tests"
      puts ""
    end
  end
end
