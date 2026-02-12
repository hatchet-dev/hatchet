# frozen_string_literal: true

require "time"

RSpec.describe Hatchet::Features::Metrics do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:task_api) { instance_double("HatchetSdkRest::TaskApi") }
  let(:tenant_api) { instance_double("HatchetSdkRest::TenantApi") }
  let(:metrics_client) { described_class.new(rest_client, config) }

  before do
    allow(HatchetSdkRest::TaskApi).to receive(:new).with(rest_client).and_return(task_api)
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
    it "creates a new metrics client with required dependencies" do
      expect(metrics_client).to be_a(described_class)
      expect(metrics_client.instance_variable_get(:@config)).to eq(config)
    end

    it "initializes API clients" do
      described_class.new(rest_client, config)
      expect(HatchetSdkRest::TaskApi).to have_received(:new).with(rest_client)
      expect(HatchetSdkRest::TenantApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#get_queue_metrics" do
    it "retrieves queue metrics" do
      result_obj = double("result", queues: { "default" => 10 })
      allow(tenant_api).to receive(:tenant_get_step_run_queue_metrics).with("test-tenant").and_return(result_obj)

      result = metrics_client.get_queue_metrics

      expect(result).to eq({ "default" => 10 })
    end

    it "returns empty hash when queues is nil" do
      result_obj = double("result", queues: nil)
      allow(tenant_api).to receive(:tenant_get_step_run_queue_metrics).with("test-tenant").and_return(result_obj)

      result = metrics_client.get_queue_metrics

      expect(result).to eq({})
    end
  end

  describe "#scrape_tenant_prometheus_metrics" do
    it "retrieves prometheus metrics" do
      prometheus_text = "# HELP hatchet_tasks_total Total tasks\nhatchet_tasks_total 42"
      allow(tenant_api).to receive(:tenant_get_prometheus_metrics).with("test-tenant").and_return(prometheus_text)

      result = metrics_client.scrape_tenant_prometheus_metrics

      expect(result).to eq(prometheus_text)
    end
  end

  describe "#get_task_stats" do
    it "retrieves task statistics" do
      stats = instance_double("Object")
      allow(tenant_api).to receive(:tenant_get_task_stats).with("test-tenant").and_return(stats)

      result = metrics_client.get_task_stats

      expect(result).to eq(stats)
    end
  end

  describe "#get_task_metrics" do
    let(:metric_completed) { double("metric", status: :completed, count: 100) }
    let(:metric_failed) { double("metric", status: :failed, count: 5) }
    let(:metric_running) { double("metric", status: :running, count: 10) }
    let(:metric_queued) { double("metric", status: :queued, count: 20) }
    let(:metric_cancelled) { double("metric", status: :cancelled, count: 2) }

    before do
      allow(task_api).to receive(:v1_task_list_status_metrics)
        .and_return([metric_completed, metric_failed, metric_running, metric_queued, metric_cancelled])
    end

    it "returns task metrics grouped by status" do
      result = metrics_client.get_task_metrics

      expect(result).to be_a(Hatchet::Features::TaskMetrics)
      expect(result.completed).to eq(100)
      expect(result.failed).to eq(5)
      expect(result.running).to eq(10)
      expect(result.queued).to eq(20)
      expect(result.cancelled).to eq(2)
    end

    it "passes custom parameters" do
      since_time = Time.now - 3600
      until_time = Time.now

      metrics_client.get_task_metrics(
        since: since_time,
        until_time: until_time,
        workflow_ids: ["wf-1"],
        parent_task_external_id: "parent-123",
        triggering_event_external_id: "event-456"
      )

      expect(task_api).to have_received(:v1_task_list_status_metrics).with(
        "test-tenant",
        since_time.utc.iso8601,
        {
          _until: until_time.utc.iso8601,
          workflow_ids: ["wf-1"],
          parent_task_external_id: "parent-123",
          triggering_event_external_id: "event-456"
        }
      )
    end

    it "defaults to zero for missing statuses" do
      allow(task_api).to receive(:v1_task_list_status_metrics).and_return([])

      result = metrics_client.get_task_metrics

      expect(result.completed).to eq(0)
      expect(result.failed).to eq(0)
      expect(result.running).to eq(0)
      expect(result.queued).to eq(0)
      expect(result.cancelled).to eq(0)
    end
  end
end
