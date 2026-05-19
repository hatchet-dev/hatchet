# frozen_string_literal: true

require "time"

RSpec.describe Hatchet::Features::Logs do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:log_api) { instance_double("HatchetSdkRest::LogApi") }
  let(:logs_client) { described_class.new(rest_client, config) }

  before do
    allow(HatchetSdkRest::LogApi).to receive(:new).with(rest_client).and_return(log_api)
  end

  around do |example|
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    example.run
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    it "creates a new logs client with required dependencies" do
      expect(logs_client).to be_a(described_class)
      expect(logs_client.instance_variable_get(:@config)).to eq(config)
      expect(logs_client.instance_variable_get(:@rest_client)).to eq(rest_client)
    end

    it "initializes log API client" do
      described_class.new(rest_client, config)
      expect(HatchetSdkRest::LogApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#list" do
    let(:task_run_id) { "task-run-123" }
    let(:log_list) { instance_double("Object") }

    before do
      allow(log_api).to receive(:v1_log_line_list).and_return(log_list)
    end

    it "lists logs with default parameters" do
      result = logs_client.list(task_run_id)

      expect(result).to eq(log_list)
      expect(log_api).to have_received(:v1_log_line_list).with(
        task_run_id,
        {
          limit: 1000,
          since: nil,
          _until: nil,
        },
      )
    end

    it "lists logs with custom parameters" do
      since_time = Time.now - 3600
      until_time = Time.now

      logs_client.list(task_run_id, limit: 500, since: since_time, until_time: until_time)

      expect(log_api).to have_received(:v1_log_line_list).with(
        task_run_id,
        {
          limit: 500,
          since: since_time.utc.iso8601,
          _until: until_time.utc.iso8601,
        },
      )
    end

    it "returns the log list" do
      result = logs_client.list(task_run_id)
      expect(result).to eq(log_list)
    end
  end
end
