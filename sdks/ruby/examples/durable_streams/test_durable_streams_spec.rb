# frozen_string_literal: true

require "json"
require "securerandom"
require "timeout"
require_relative "../spec_helper"
require_relative "worker"

RSpec.describe "DurableStreams" do
  def scope(name)
    "#{name}_#{SecureRandom.uuid}"
  end

  def collect(topic, count, **opts)
    events = []
    Timeout.timeout(30) do
      HATCHET.streams.subscribe(topic, **opts) do |event|
        events << event
        break if events.length >= count
      end
    end
    events
  end

  it "reads from the beginning, then live" do
    topic = scope("durable_streams_from_beginning")
    HATCHET.streams.publish(topic, "published-before-consume")

    publisher = Thread.new do
      sleep 0.2
      %w[live-1 live-2].each { |m| HATCHET.streams.publish(topic, m) }
    end

    expect(collect(topic, 3).map(&:payload)).to eq(%w[published-before-consume live-1 live-2])
    publisher.join
  end

  it "resumes from a cursor" do
    topic = scope("durable_streams_resume_cursor")
    %w[one two three four].each { |m| HATCHET.streams.publish(topic, m) }

    first_two = collect(topic, 2)
    expect(first_two.map(&:payload)).to eq(%w[one two])
    expect(collect(topic, 2, cursor: first_two.last.cursor).map(&:payload)).to eq(%w[three four])
  end

  it "starts new readers after a reset" do
    topic = scope("durable_streams_reset")
    %w[one two three].each { |m| HATCHET.streams.publish(topic, m) }

    first = collect(topic, 1).first
    HATCHET.streams.reset(topic, cursor: first.cursor)
    expect(collect(topic, 2).map(&:payload)).to eq(%w[two three])

    HATCHET.streams.reset(topic)
    HATCHET.streams.publish(topic, "after-reset")
    expect(collect(topic, 1).map(&:payload)).to eq(["after-reset"])
  end

  it "isolates namespaces" do
    topic = scope("durable_streams_namespace")
    HATCHET.streams.publish(topic, "from-a", namespace: "ns-a")
    HATCHET.streams.publish(topic, "from-b", namespace: "ns-b")

    expect(collect(topic, 1, namespace: "ns-b").map(&:payload)).to eq(["from-b"])
  end

  it "describes a topic" do
    topic = scope("durable_streams_metadata")
    %w[first second third].each { |m| HATCHET.streams.publish(topic, m) }

    metadata = HATCHET.streams.topic_metadata(topic)
    expect(metadata.topic).to eq(topic)
    expect(metadata.message_count).to eq(3)
    expect(metadata.last_published_at).to be_a(Time)
    expect(metadata.latest_cursor).to eq(collect(topic, 3).last.cursor)

    expect { HATCHET.streams.topic_metadata(scope("durable_streams_missing")) }.to raise_error(GRPC::NotFound)
  end

  it "round-trips a payload too large for one gRPC message" do
    topic = scope("durable_streams_large_payload")
    large = "x" * (5 * 1024 * 1024)

    HATCHET.streams.publish(topic, large)
    HATCHET.streams.publish(topic, "small")

    events = collect(topic, 2)
    expect(events[0].payload).to eq(large)
    expect(events[1].payload).to eq("small")
  end

  it "skips a failed attempt's messages for new readers" do
    topic = scope("durable_streams_retry")
    DURABLE_STREAMS_RETRYING_PROGRESS_TASK.run({ "topic" => topic })

    received = []
    HATCHET.streams.subscribe(topic) do |event|
      message = JSON.parse(event.payload)
      break if message["done"]

      received << message
    end

    expect(received).to eq((0..2).map { |chunk| { "attempt" => 1, "chunk" => chunk } })
  end
end
