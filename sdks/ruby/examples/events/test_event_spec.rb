# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "EventWorkflow" do
  it "pushes an event" do
    e = hatchet.events.create(key: EVENT_KEY, data: { "should_skip" => false })
    expect(e).not_to be_nil
  end

  it "bulk pushes events" do
    events = [
      { key: "event1", payload: { "message" => "Event 1", "should_skip" => false },
        additional_metadata: { "source" => "test", "user_id" => "user123" } },
      { key: "event2", payload: { "message" => "Event 2", "should_skip" => false },
        additional_metadata: { "source" => "test", "user_id" => "user456" } },
      { key: "event3", payload: { "message" => "Event 3", "should_skip" => false },
        additional_metadata: { "source" => "test", "user_id" => "user789" } }
    ]

    result = hatchet.events.bulk_push(events)
    expect(result.length).to eq(3)
  end
end
