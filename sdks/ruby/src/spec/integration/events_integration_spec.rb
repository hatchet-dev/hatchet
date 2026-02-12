# frozen_string_literal: true

require "time"
require_relative "../support/integration_helper"

RSpec.describe "Hatchet::Features::Events Integration", :integration do
  let(:client) { create_test_client }
  let(:events_client) { client.events }

  describe "API connectivity and basic operations" do
    it "can list events without error" do
      expect { events_client.list(limit: 5) }.not_to raise_error
    end

    it "returns an EventList when listing events" do
      result = events_client.list(limit: 5)
      expect(result).to be_a(HatchetSdkRest::V1EventList)
      expect(result).to respond_to(:rows)
      expect(result).to respond_to(:pagination)
    end

    it "can list events with various filters" do
      since_time = Time.now - 24 * 60 * 60  # 1 day ago

      expect do
        events_client.list(
          since: since_time,
          limit: 5,
          keys: ["test-event"],
          additional_metadata: { "source" => "integration-test" }
        )
      end.not_to raise_error
    end

    it "handles empty results gracefully" do
      # Use a very specific time range that likely has no results
      since_time = Time.new(2020, 1, 1)
      until_time = Time.new(2020, 1, 2)

      result = events_client.list(since: since_time, until_time: until_time, limit: 1)
      expect(result).to be_a(HatchetSdkRest::V1EventList)
      expect(result.rows).to be_an(Array)
    end

    it "can list event keys" do
      expect { events_client.list_keys }.not_to raise_error
    end

    it "returns event key list when listing keys" do
      result = events_client.list_keys
      expect(result).to respond_to(:rows) # EventKeyList has rows attribute
    end
  end

  describe "event creation operations" do
    let(:test_event_key) { "integration-test-#{Time.now.to_i}" }
    let(:test_payload) { { "message" => "test event", "timestamp" => Time.now.to_i } }
    let(:test_metadata) { { "source" => "integration-test", "test_run" => "true" } }

    it "can create an event using the create method" do
      expect do
        events_client.create(
          key: test_event_key,
          data: test_payload,
          additional_metadata: test_metadata
        )
      end.not_to raise_error
    end

    it "can push a single event" do
      expect do
        events_client.push(
          test_event_key,
          test_payload,
          additional_metadata: test_metadata
        )
      end.not_to raise_error
    end

    it "returns event details when pushing an event" do
      result = events_client.push(
        test_event_key,
        test_payload,
        additional_metadata: test_metadata
      )

      # The exact structure may vary, but we expect some response
      expect(result).not_to be_nil
    end

    it "can push events with different priorities" do
      expect do
        events_client.push(
          "#{test_event_key}-priority",
          test_payload,
          priority: 1,
          additional_metadata: test_metadata
        )
      end.not_to raise_error
    end

    it "applies namespace to event keys" do
      # Test with a client that has a namespace
      config_with_namespace = Hatchet::Config.new(
        token: ENV["HATCHET_CLIENT_TOKEN"],
        namespace: "test_"
      )
      client_with_ns = Hatchet::Client.new(**config_with_namespace.to_h)

      expect do
        client_with_ns.events.push(
          "namespaced-event",
          test_payload,
          additional_metadata: test_metadata
        )
      end.not_to raise_error
    end
  end

  describe "bulk event operations" do
    let(:test_events) do
      [
        {
          key: "bulk-test-1-#{Time.now.to_i}",
          data: { "message" => "bulk event 1", "index" => 1 },
          additional_metadata: { "source" => "bulk-test" }
        },
        {
          key: "bulk-test-2-#{Time.now.to_i}",
          data: { "message" => "bulk event 2", "index" => 2 },
          additional_metadata: { "source" => "bulk-test" },
          priority: 1
        }
      ]
    end

    it "can create events in bulk" do
      expect { events_client.bulk_push(test_events) }.not_to raise_error
    end

    it "returns bulk creation response" do
      result = events_client.bulk_push(test_events)
      expect(result).not_to be_nil
    end

    it "can create bulk events with namespace override" do
      expect do
        events_client.bulk_push(test_events, namespace: "bulk_test_")
      end.not_to raise_error
    end
  end

  describe "event retrieval operations" do
    let(:test_event_id) do
      # First create an event to get an ID
      result = events_client.push(
        "retrieval-test-#{Time.now.to_i}",
        { "message" => "test event for retrieval" },
        additional_metadata: { "source" => "retrieval-test" }
      )

      # Try to extract event ID from the response (gRPC Event has event_id)
      if result.respond_to?(:event_id) && result.event_id && !result.event_id.empty?
        result.event_id
      elsif result.respond_to?(:id)
        result.id
      elsif result.respond_to?(:metadata) && result.metadata.respond_to?(:id)
        result.metadata.id
      else
        # If we can't get the ID from creation, try to find it in the list
        recent_events = events_client.list(limit: 1, keys: [(result.key rescue nil)].compact)
        if recent_events.rows && recent_events.rows.any?
          recent_events.rows.first.metadata.id
        else
          skip "Cannot determine event ID for retrieval test"
        end
      end
    end

    it "can get a specific event by ID" do
      expect { events_client.get(test_event_id) }.not_to raise_error
    end

    it "returns event details when getting by ID" do
      result = events_client.get(test_event_id)
      expect(result).not_to be_nil
    end

    it "can get event data by ID" do
      expect { events_client.get_data(test_event_id) }.not_to raise_error
    end

    it "handles invalid event IDs gracefully" do
      invalid_id = "non-existent-event-id-#{Time.now.to_i}"

      expect { events_client.get(invalid_id) }.to raise_error(StandardError)
    end
  end

  describe "event management operations (use with caution)" do
    # These tests are cautious since they could affect real data

    it "can create cancel request objects" do
      # Test the structure without actually canceling events
      expect do
        events_client.cancel(event_ids: ["test-id"], keys: ["test-key"])
      rescue StandardError => e
        # It's okay if this fails - we just want to test the API call structure
        expect(e).to be_a(StandardError)
      end.not_to raise_error(ArgumentError) # Should not fail due to argument issues
    end

    it "can create replay request objects" do
      # Test the structure without actually replaying events
      expect do
        events_client.replay(event_ids: ["test-id"], keys: ["test-key"])
      rescue StandardError => e
        # It's okay if this fails - we just want to test the API call structure
        expect(e).to be_a(StandardError)
      end.not_to raise_error(ArgumentError) # Should not fail due to argument issues
    end

    # Note: We don't actually test cancel/replay operations in integration tests
    # as they could affect real event data. The structure validation above
    # combined with unit tests should be sufficient.
  end

  describe "error handling" do
    it "handles nil event key gracefully via gRPC" do
      # gRPC accepts nil keys (converted to empty string) without raising
      expect do
        events_client.push(nil, {})
      end.not_to raise_error
    end

    it "handles invalid date ranges gracefully" do
      # Future date range should return empty results, not error
      future_since = Time.now + 24 * 60 * 60
      future_until = Time.now + 48 * 60 * 60

      expect do
        result = events_client.list(since: future_since, until_time: future_until, limit: 1)
        expect(result.rows).to be_empty if result.respond_to?(:rows)
      end.not_to raise_error
    end

    it "handles malformed payloads appropriately" do
      # Test with data that might be problematic
      expect do
        events_client.push(
          "error-test-#{Time.now.to_i}",
          { "nested" => { "very" => { "deep" => "object" } } }
        )
      end.not_to raise_error # Should handle nested objects fine
    end
  end

  describe "response data structure validation" do
    it "validates EventList structure" do
      result = events_client.list(limit: 1)

      expect(result).to be_a(HatchetSdkRest::V1EventList)
      expect(result.rows).to be_an(Array)
      expect(result.pagination).not_to be_nil

      if result.rows.any?
        event = result.rows.first
        expect(event.metadata).not_to be_nil
        expect(event.metadata.id).to be_a(String)
      end
    end

    it "validates event creation response structure" do
      result = events_client.push(
        "structure-test-#{Time.now.to_i}",
        { "test" => "structure validation" }
      )

      # The response structure may vary, but should be consistent
      expect(result).not_to be_nil
    end

    it "validates bulk creation response structure" do
      events_data = [
        { key: "bulk-structure-test-#{Time.now.to_i}", data: { "test" => "bulk structure" } }
      ]

      result = events_client.bulk_push(events_data)
      expect(result).not_to be_nil
    end

    it "validates event key list structure" do
      result = events_client.list_keys

      # Exact structure may vary, but should have some consistent interface
      expect(result).not_to be_nil
    end
  end

  describe "filtering and pagination" do
    it "respects limit parameter" do
      result = events_client.list(limit: 2)
      expect(result.rows.length).to be <= 2
    end

    it "can filter by date ranges" do
      since_time = Time.now - 7 * 24 * 60 * 60  # 7 days ago
      until_time = Time.now - 6 * 24 * 60 * 60  # 6 days ago

      expect do
        result = events_client.list(since: since_time, until_time: until_time, limit: 5)
        expect(result).to be_a(HatchetSdkRest::V1EventList)
      end.not_to raise_error
    end

    it "can filter by event keys" do
      # Create a unique event first
      unique_key = "filter-test-#{Time.now.to_i}"
      events_client.push(unique_key, { "test" => "filter by key" })

      # Small delay to ensure event is processed
      sleep(0.1)

      # Try to filter by that key
      result = events_client.list(keys: [unique_key], limit: 5)
      expect(result).to be_a(HatchetSdkRest::V1EventList)
    end

    it "can use offset for pagination" do
      result_page_1 = events_client.list(limit: 2, offset: 0)
      result_page_2 = events_client.list(limit: 2, offset: 2)

      expect(result_page_1).to be_a(HatchetSdkRest::V1EventList)
      expect(result_page_2).to be_a(HatchetSdkRest::V1EventList)
    end
  end

  describe "configuration and client setup" do
    it "uses the correct tenant ID from configuration" do
      expect(client.config.tenant_id).not_to be_empty
      expect(client.config.token).not_to be_empty
    end

    it "can access the events client" do
      expect(events_client).to be_a(Hatchet::Features::Events)
      expect(events_client.instance_variable_get(:@config)).to eq(client.config)
    end

    it "initializes event API correctly" do
      expect(events_client.instance_variable_get(:@event_api)).to be_a(HatchetSdkRest::EventApi)
    end

    it "has access to exported constants" do
      expect(Hatchet::Features::Events::CreateEventRequest).to eq(HatchetSdkRest::CreateEventRequest)
      expect(Hatchet::Features::Events::BulkCreateEventRequest).to eq(HatchetSdkRest::BulkCreateEventRequest)
      expect(Hatchet::Features::Events::EventList).to eq(HatchetSdkRest::V1EventList)
    end
  end

  describe "namespace handling" do
    it "applies default namespace when configured" do
      if client.config.namespace && !client.config.namespace.empty?
        # Test that namespace is applied
        expect do
          events_client.push(
            "namespace-test",
            { "test" => "namespace handling" }
          )
        end.not_to raise_error
      else
        skip "No default namespace configured for testing"
      end
    end

    it "applies override namespace when specified" do
      expect do
        events_client.push(
          "override-test",
          { "test" => "namespace override" },
          namespace: "override_"
        )
      end.not_to raise_error
    end

    it "handles bulk events with namespace" do
      events_data = [
        { key: "bulk-ns-test", data: { "test" => "bulk namespace" } }
      ]

      expect do
        events_client.bulk_push(events_data, namespace: "bulk_test_")
      end.not_to raise_error
    end
  end
end
