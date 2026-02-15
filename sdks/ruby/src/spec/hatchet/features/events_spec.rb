# frozen_string_literal: true

require "time"

RSpec.describe Hatchet::Features::Events do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0LXRlbmFudCJ9.signature" }
  let(:config) { Hatchet::Config.new(token: valid_token) }
  let(:rest_client) { instance_double("ApiClient") }
  let(:event_grpc) { instance_double("Hatchet::Clients::Grpc::EventClient") }
  let(:event_api) { instance_double("HatchetSdkRest::EventApi") }
  let(:events_client) { described_class.new(rest_client, event_grpc, config) }

  before do
    allow(HatchetSdkRest::EventApi).to receive(:new).with(rest_client).and_return(event_api)
  end

  around do |example|
    # Store original env vars
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }

    # Clear all HATCHET_CLIENT_ env vars before each test
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }

    example.run

    # Restore original env vars
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    it "creates a new events client with required dependencies" do
      expect(events_client).to be_a(described_class)
      expect(events_client.instance_variable_get(:@config)).to eq(config)
      expect(events_client.instance_variable_get(:@rest_client)).to eq(rest_client)
      expect(events_client.instance_variable_get(:@event_grpc)).to eq(event_grpc)
    end

    it "initializes event API client" do
      described_class.new(rest_client, event_grpc, config)
      expect(HatchetSdkRest::EventApi).to have_received(:new).with(rest_client)
    end
  end

  describe "#create" do
    let(:event_key) { "test-event" }
    let(:event_data) { { "message" => "test" } }
    let(:additional_metadata) { { "source" => "test" } }
    let(:priority) { 1 }
    let(:scope) { "test-scope" }
    let(:grpc_response) { instance_double("Object") }

    before do
      allow(event_grpc).to receive(:push).and_return(grpc_response)
    end

    it "creates an event with required parameters" do
      result = events_client.create(key: event_key, data: event_data)

      expect(result).to eq(grpc_response)
      expect(event_grpc).to have_received(:push).with(
        key: "test-event",
        payload: event_data,
        additional_metadata: nil,
        priority: nil,
        scope: nil,
        namespace: nil,
      )
    end

    it "creates an event with all optional parameters" do
      result = events_client.create(
        key: event_key,
        data: event_data,
        additional_metadata: additional_metadata,
        priority: priority,
        scope: scope,
      )

      expect(result).to eq(grpc_response)
      expect(event_grpc).to have_received(:push).with(
        key: "test-event",
        payload: event_data,
        additional_metadata: additional_metadata,
        priority: priority,
        scope: scope,
        namespace: nil,
      )
    end
  end

  describe "#push" do
    let(:event_key) { "test-event" }
    let(:payload) { { "message" => "test" } }
    let(:additional_metadata) { { "source" => "test" } }
    let(:grpc_response) { instance_double("Object") }

    before do
      allow(event_grpc).to receive(:push).and_return(grpc_response)
    end

    it "pushes a single event with basic parameters" do
      events_client.push(event_key, payload)

      expect(event_grpc).to have_received(:push).with(
        key: event_key,
        payload: payload,
        additional_metadata: nil,
        priority: nil,
        scope: nil,
        namespace: nil,
      )
    end

    it "pushes a single event with all parameters" do
      events_client.push(event_key, payload, additional_metadata: additional_metadata, priority: 1)

      expect(event_grpc).to have_received(:push).with(
        key: event_key,
        payload: payload,
        additional_metadata: additional_metadata,
        priority: 1,
        scope: nil,
        namespace: nil,
      )
    end

    it "applies namespace via gRPC client" do
      events_client.push(event_key, payload)

      expect(event_grpc).to have_received(:push).with(
        key: event_key,
        payload: payload,
        additional_metadata: nil,
        priority: nil,
        scope: nil,
        namespace: nil,
      )
    end

    it "passes namespace override to gRPC client" do
      events_client.push(event_key, payload, namespace: "override_")

      expect(event_grpc).to have_received(:push).with(
        key: event_key,
        payload: payload,
        additional_metadata: nil,
        priority: nil,
        scope: nil,
        namespace: "override_",
      )
    end

    it "returns the gRPC response" do
      result = events_client.push(event_key, payload)
      expect(result).to eq(grpc_response)
    end
  end

  describe "#bulk_push" do
    let(:events_data) do
      [
        { key: "event-1", data: { "message" => "first" } },
        { key: "event-2", data: { "message" => "second" }, additional_metadata: { "type" => "test" }, priority: 1 },
      ]
    end
    let(:grpc_response) { instance_double("Object") }

    before do
      allow(event_grpc).to receive(:bulk_push).and_return(grpc_response)
    end

    it "creates bulk events" do
      events_client.bulk_push(events_data)

      expect(event_grpc).to have_received(:bulk_push).with(
        [
          { key: "event-1", payload: { "message" => "first" }, additional_metadata: nil, priority: nil },
          { key: "event-2", payload: { "message" => "second" }, additional_metadata: { "type" => "test" }, priority: 1 },
        ],
        namespace: nil,
      )
    end

    it "passes namespace to gRPC client" do
      events_client.bulk_push(events_data, namespace: "bulk_")

      expect(event_grpc).to have_received(:bulk_push).with(
        [
          { key: "event-1", payload: { "message" => "first" }, additional_metadata: nil, priority: nil },
          { key: "event-2", payload: { "message" => "second" }, additional_metadata: { "type" => "test" }, priority: 1 },
        ],
        namespace: "bulk_",
      )
    end

    it "handles events with missing data" do
      events_data = [{ key: "event-1" }]

      events_client.bulk_push(events_data)

      expect(event_grpc).to have_received(:bulk_push).with(
        [{ key: "event-1", payload: {}, additional_metadata: nil, priority: nil }],
        namespace: nil,
      )
    end

    it "returns the gRPC response" do
      result = events_client.bulk_push(events_data)
      expect(result).to eq(grpc_response)
    end
  end

  describe "#list" do
    let(:event_list) { instance_double("HatchetSdkRest::V1EventList") }
    let(:since_time) { Time.now - 3600 }
    let(:until_time) { Time.now }

    before do
      allow(event_api).to receive(:v1_event_list).and_return(event_list)
    end

    it "lists events with default parameters" do
      events_client.list

      expect(event_api).to have_received(:v1_event_list).with(
        "test-tenant",
        {
          offset: nil,
          limit: nil,
          keys: nil,
          since: nil,
          until: nil,
          workflow_ids: nil,
          workflow_run_statuses: nil,
          event_ids: nil,
          additional_metadata: nil,
          scopes: nil,
        },
      )
    end

    it "lists events with all parameters" do
      events_client.list(
        offset: 10,
        limit: 50,
        keys: %w[event-1 event-2],
        since: since_time,
        until_time: until_time,
        workflow_ids: ["workflow-1"],
        workflow_run_statuses: ["RUNNING"],
        event_ids: ["event-id-1"],
        additional_metadata: { "source" => "test" },
        scopes: ["scope-1"],
      )

      expect(event_api).to have_received(:v1_event_list).with(
        "test-tenant",
        {
          offset: 10,
          limit: 50,
          keys: %w[event-1 event-2],
          since: since_time.utc.iso8601,
          until: until_time.utc.iso8601,
          workflow_ids: ["workflow-1"],
          workflow_run_statuses: ["RUNNING"],
          event_ids: ["event-id-1"],
          additional_metadata: [{ key: "source", value: "test" }],
          scopes: ["scope-1"],
        },
      )
    end

    it "returns the event list" do
      result = events_client.list
      expect(result).to eq(event_list)
    end
  end

  describe "#get" do
    let(:event_id) { "event-123" }
    let(:event_details) { instance_double("Object") }

    it "gets event by ID" do
      allow(event_api).to receive(:event_get).with(event_id).and_return(event_details)

      result = events_client.get(event_id)

      expect(result).to eq(event_details)
      expect(event_api).to have_received(:event_get).with(event_id)
    end
  end

  describe "#get_data" do
    let(:event_id) { "event-123" }
    let(:event_data) { instance_double("Object") }

    it "gets event data by ID" do
      allow(event_api).to receive(:event_data_get).with(event_id).and_return(event_data)

      result = events_client.get_data(event_id)

      expect(result).to eq(event_data)
      expect(event_api).to have_received(:event_data_get).with(event_id)
    end
  end

  describe "#list_keys" do
    let(:event_keys) { instance_double("Object") }

    it "lists event keys" do
      allow(event_api).to receive(:v1_event_key_list).with("test-tenant").and_return(event_keys)

      result = events_client.list_keys

      expect(result).to eq(event_keys)
      expect(event_api).to have_received(:v1_event_key_list).with("test-tenant")
    end
  end

  describe "#cancel" do
    let(:cancel_request) { instance_double("HatchetSdkRest::CancelEventRequest") }
    let(:cancel_response) { instance_double("Object") }
    let(:since_time) { Time.now - 3600 }
    let(:until_time) { Time.now }

    before do
      allow(HatchetSdkRest::CancelEventRequest).to receive(:new).and_return(cancel_request)
      allow(event_api).to receive(:event_update_cancel).and_return(cancel_response)
    end

    it "cancels events with event IDs" do
      events_client.cancel(event_ids: %w[event-1 event-2])

      expect(HatchetSdkRest::CancelEventRequest).to have_received(:new).with(
        event_ids: %w[event-1 event-2],
        keys: nil,
        since: nil,
        until: nil,
      )
      expect(event_api).to have_received(:event_update_cancel).with("test-tenant", cancel_request)
    end

    it "cancels events with keys and date range" do
      events_client.cancel(keys: ["event-key"], since: since_time, until_time: until_time)

      expect(HatchetSdkRest::CancelEventRequest).to have_received(:new).with(
        event_ids: nil,
        keys: ["event-key"],
        since: since_time.utc.iso8601,
        until: until_time.utc.iso8601,
      )
    end

    it "returns the cancel response" do
      result = events_client.cancel(event_ids: ["event-1"])
      expect(result).to eq(cancel_response)
    end
  end

  describe "#replay" do
    let(:replay_request) { instance_double("HatchetSdkRest::ReplayEventRequest") }
    let(:replay_response) { instance_double("Object") }
    let(:since_time) { Time.now - 3600 }
    let(:until_time) { Time.now }

    before do
      allow(HatchetSdkRest::ReplayEventRequest).to receive(:new).and_return(replay_request)
      allow(event_api).to receive(:event_update_replay).and_return(replay_response)
    end

    it "replays events with event IDs" do
      events_client.replay(event_ids: %w[event-1 event-2])

      expect(HatchetSdkRest::ReplayEventRequest).to have_received(:new).with(
        event_ids: %w[event-1 event-2],
        keys: nil,
        since: nil,
        until: nil,
      )
      expect(event_api).to have_received(:event_update_replay).with("test-tenant", replay_request)
    end

    it "replays events with keys and date range" do
      events_client.replay(keys: ["event-key"], since: since_time, until_time: until_time)

      expect(HatchetSdkRest::ReplayEventRequest).to have_received(:new).with(
        event_ids: nil,
        keys: ["event-key"],
        since: since_time.utc.iso8601,
        until: until_time.utc.iso8601,
      )
    end

    it "returns the replay response" do
      result = events_client.replay(event_ids: ["event-1"])
      expect(result).to eq(replay_response)
    end
  end

  describe "private methods" do
    describe "#apply_namespace" do
      it "applies default namespace from config" do
        config_with_namespace = Hatchet::Config.new(token: valid_token, namespace: "test_")
        events_client_with_ns = described_class.new(rest_client, event_grpc, config_with_namespace)

        result = events_client_with_ns.send(:apply_namespace, "event-key")
        expect(result).to eq("test_event-key")
      end

      it "applies namespace override" do
        result = events_client.send(:apply_namespace, "event-key", "override_")
        expect(result).to eq("override_event-key")
      end

      it "returns event key unchanged when no namespace" do
        result = events_client.send(:apply_namespace, "event-key")
        expect(result).to eq("event-key")
      end
    end

    describe "#maybe_additional_metadata_to_kv" do
      it "converts hash to key-value array" do
        metadata = { "env" => "test", "version" => "1.0" }

        result = events_client.send(:maybe_additional_metadata_to_kv, metadata)

        expect(result).to eq([
          { key: "env", value: "test" },
          { key: "version", value: "1.0" },
        ])
      end

      it "returns nil for nil input" do
        result = events_client.send(:maybe_additional_metadata_to_kv, nil)
        expect(result).to be_nil
      end

      it "converts keys and values to strings" do
        metadata = { 123 => 456 }

        result = events_client.send(:maybe_additional_metadata_to_kv, metadata)

        expect(result).to eq([{ key: "123", value: "456" }])
      end
    end
  end

  describe "constants and exports" do
    it "exports CreateEventRequest" do
      expect(described_class::CreateEventRequest).to eq(HatchetSdkRest::CreateEventRequest)
    end

    it "exports BulkCreateEventRequest" do
      expect(described_class::BulkCreateEventRequest).to eq(HatchetSdkRest::BulkCreateEventRequest)
    end

    it "exports EventList" do
      expect(described_class::EventList).to eq(HatchetSdkRest::V1EventList)
    end
  end
end
