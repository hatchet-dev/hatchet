# frozen_string_literal: true

RSpec.describe Hatchet::Features::Workers do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:worker_api) { instance_double("HatchetSdkRest::WorkerApi") }
  let(:workers_client) { described_class.new(rest_client, config) }

  before do
    allow(HatchetSdkRest::WorkerApi).to receive(:new).with(rest_client).and_return(worker_api)
  end

  around do |example|
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    example.run
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    it "creates a new workers client with required dependencies" do
      expect(workers_client).to be_a(described_class)
      expect(workers_client.instance_variable_get(:@config)).to eq(config)
      expect(workers_client.instance_variable_get(:@rest_client)).to eq(rest_client)
    end

    it "initializes worker API client" do
      described_class.new(rest_client, config)
      expect(HatchetSdkRest::WorkerApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#get" do
    let(:worker_id) { "worker-123" }
    let(:worker_details) { instance_double("Object") }

    it "retrieves a worker by ID" do
      allow(worker_api).to receive(:worker_get).with(worker_id).and_return(worker_details)

      result = workers_client.get(worker_id)

      expect(result).to eq(worker_details)
      expect(worker_api).to have_received(:worker_get).with(worker_id)
    end
  end

  describe "#list" do
    let(:worker_list) { instance_double("Object") }

    it "lists all workers in the tenant" do
      allow(worker_api).to receive(:worker_list).with("test-tenant").and_return(worker_list)

      result = workers_client.list

      expect(result).to eq(worker_list)
      expect(worker_api).to have_received(:worker_list).with("test-tenant")
    end
  end

  describe "#update" do
    let(:worker_id) { "worker-123" }
    let(:update_request) { instance_double("HatchetSdkRest::UpdateWorkerRequest") }
    let(:updated_worker) { instance_double("Object") }

    it "updates a worker by ID" do
      opts = { is_paused: true }
      allow(HatchetSdkRest::UpdateWorkerRequest).to receive(:new).with(opts).and_return(update_request)
      allow(worker_api).to receive(:worker_update).with(worker_id, update_request).and_return(updated_worker)

      result = workers_client.update(worker_id, opts)

      expect(result).to eq(updated_worker)
      expect(HatchetSdkRest::UpdateWorkerRequest).to have_received(:new).with(opts)
      expect(worker_api).to have_received(:worker_update).with(worker_id, update_request)
    end
  end
end
