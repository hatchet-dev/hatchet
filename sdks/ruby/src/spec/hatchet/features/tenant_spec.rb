# frozen_string_literal: true

RSpec.describe Hatchet::Features::Tenant do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:tenant_api) { instance_double("HatchetSdkRest::TenantApi") }
  let(:tenant_client) { described_class.new(rest_client, config) }

  before do
    allow(HatchetSdkRest::TenantApi).to receive(:new).with(rest_client).and_return(tenant_api)
  end

  around do |example|
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    example.run
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    it "creates a new tenant client with required dependencies" do
      expect(tenant_client).to be_a(described_class)
      expect(tenant_client.instance_variable_get(:@config)).to eq(config)
      expect(tenant_client.instance_variable_get(:@rest_client)).to eq(rest_client)
    end

    it "initializes tenant API client" do
      described_class.new(rest_client, config)
      expect(HatchetSdkRest::TenantApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#get" do
    let(:tenant_details) { instance_double("Object") }

    it "retrieves the current tenant" do
      allow(tenant_api).to receive(:tenant_get).with("test-tenant").and_return(tenant_details)

      result = tenant_client.get

      expect(result).to eq(tenant_details)
      expect(tenant_api).to have_received(:tenant_get).with("test-tenant")
    end
  end
end
