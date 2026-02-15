# frozen_string_literal: true

RSpec.describe Hatchet::Features::Filters do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:filter_api) { instance_double("HatchetSdkRest::FilterApi") }
  let(:filters_client) { described_class.new(rest_client, config) }

  before do
    allow(HatchetSdkRest::FilterApi).to receive(:new).with(rest_client).and_return(filter_api)
  end

  around do |example|
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    example.run
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    it "creates a new filters client with required dependencies" do
      expect(filters_client).to be_a(described_class)
      expect(filters_client.instance_variable_get(:@config)).to eq(config)
    end

    it "initializes filter API client" do
      described_class.new(rest_client, config)
      expect(HatchetSdkRest::FilterApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#list" do
    let(:filter_list) { instance_double("Object") }

    before do
      allow(filter_api).to receive(:v1_filter_list).and_return(filter_list)
    end

    it "lists filters with default parameters" do
      result = filters_client.list

      expect(result).to eq(filter_list)
      expect(filter_api).to have_received(:v1_filter_list).with(
        "test-tenant",
        { limit: nil, offset: nil, workflow_ids: nil, scopes: nil },
      )
    end

    it "lists filters with custom parameters" do
      filters_client.list(limit: 10, offset: 5, workflow_ids: ["wf-1"], scopes: ["scope-1"])

      expect(filter_api).to have_received(:v1_filter_list).with(
        "test-tenant",
        { limit: 10, offset: 5, workflow_ids: ["wf-1"], scopes: ["scope-1"] },
      )
    end
  end

  describe "#get" do
    let(:filter_id) { "filter-123" }
    let(:filter_details) { instance_double("Object") }

    it "retrieves a filter by ID" do
      allow(filter_api).to receive(:v1_filter_get).with("test-tenant", filter_id).and_return(filter_details)

      result = filters_client.get(filter_id)

      expect(result).to eq(filter_details)
    end
  end

  describe "#create" do
    let(:create_request) { instance_double("HatchetSdkRest::V1CreateFilterRequest") }
    let(:created_filter) { instance_double("Object") }

    before do
      allow(HatchetSdkRest::V1CreateFilterRequest).to receive(:new).and_return(create_request)
      allow(filter_api).to receive(:v1_filter_create).and_return(created_filter)
    end

    it "creates a filter with required parameters" do
      result = filters_client.create(
        workflow_id: "wf-1",
        expression: "input.value > 10",
        scope: "my-scope",
      )

      expect(result).to eq(created_filter)
      expect(HatchetSdkRest::V1CreateFilterRequest).to have_received(:new).with(
        workflow_id: "wf-1",
        expression: "input.value > 10",
        scope: "my-scope",
        payload: nil,
      )
      expect(filter_api).to have_received(:v1_filter_create).with("test-tenant", create_request)
    end

    it "creates a filter with payload" do
      filters_client.create(
        workflow_id: "wf-1",
        expression: "input.value > 10",
        scope: "my-scope",
        payload: { threshold: 10 },
      )

      expect(HatchetSdkRest::V1CreateFilterRequest).to have_received(:new).with(
        workflow_id: "wf-1",
        expression: "input.value > 10",
        scope: "my-scope",
        payload: { threshold: 10 },
      )
    end
  end

  describe "#delete" do
    let(:filter_id) { "filter-123" }
    let(:deleted_filter) { instance_double("Object") }

    it "deletes a filter by ID" do
      allow(filter_api).to receive(:v1_filter_delete).with("test-tenant", filter_id).and_return(deleted_filter)

      result = filters_client.delete(filter_id)

      expect(result).to eq(deleted_filter)
    end
  end

  describe "#update" do
    let(:filter_id) { "filter-123" }
    let(:update_request) { instance_double("HatchetSdkRest::V1UpdateFilterRequest") }
    let(:updated_filter) { instance_double("Object") }

    it "updates a filter by ID" do
      updates = { expression: "input.value > 20" }
      allow(HatchetSdkRest::V1UpdateFilterRequest).to receive(:new).with(updates).and_return(update_request)
      allow(filter_api).to receive(:v1_filter_update).with("test-tenant", filter_id, update_request).and_return(updated_filter)

      result = filters_client.update(filter_id, updates)

      expect(result).to eq(updated_filter)
    end
  end
end
