# frozen_string_literal: true

RSpec.describe Hatchet::Features::RateLimits do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:admin_grpc) { instance_double("Hatchet::Clients::Grpc::Admin") }
  let(:rate_limits_client) { described_class.new(admin_grpc, config) }

  around do |example|
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    example.run
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    it "creates a new rate limits client with required dependencies" do
      expect(rate_limits_client).to be_a(described_class)
      expect(rate_limits_client.instance_variable_get(:@admin_grpc)).to eq(admin_grpc)
      expect(rate_limits_client.instance_variable_get(:@config)).to eq(config)
    end
  end

  describe "#put" do
    it "puts a rate limit with default duration" do
      allow(admin_grpc).to receive(:put_rate_limit)

      rate_limits_client.put(key: "api-calls", limit: 100)

      expect(admin_grpc).to have_received(:put_rate_limit).with(
        key: "api-calls",
        limit: 100,
        duration: :SECOND,
      )
    end

    it "puts a rate limit with custom duration" do
      allow(admin_grpc).to receive(:put_rate_limit)

      rate_limits_client.put(key: "api-calls", limit: 1000, duration: :MINUTE)

      expect(admin_grpc).to have_received(:put_rate_limit).with(
        key: "api-calls",
        limit: 1000,
        duration: :MINUTE,
      )
    end
  end
end
