# frozen_string_literal: true

require "securerandom"
require_relative "worker"

def consume
  # > Consume
  topic = "progress-#{SecureRandom.uuid}"
  DURABLE_STREAMS_PROGRESS_TASK.run_no_wait({ "topic" => topic })

  HATCHET.streams.subscribe(topic) do |event|
    message = JSON.parse(event.payload)
    puts "step #{message['step']} of #{message['total']}"

    break if message["step"] == message["total"]
  end
end

def publish_from_anywhere
  # > Publish from anywhere
  HATCHET.streams.publish("orders", JSON.generate({ "order_id" => "ord_123", "status" => "shipped" }))
end

def resume_from_cursor(topic)
  # > Resume from a cursor
  cursor = nil

  HATCHET.streams.subscribe(topic) do |event|
    puts event.payload
    cursor = event.cursor
    break
  end

  HATCHET.streams.subscribe(topic, cursor: cursor) do |event|
    puts event.payload
  end
end

def stop_reading(topic)
  # > Stop reading
  deadline = Time.now + 10

  HATCHET.streams.subscribe(topic) do |event|
    puts event.payload
    break if Time.now > deadline
  end
end

def namespaces
  # > Namespaces
  HATCHET.streams.publish("updates", "hello", namespace: "customer-a")

  HATCHET.streams.subscribe("updates", namespace: "customer-a") do |event|
    puts event.payload
    break
  end
end

def reset_topic
  # > Reset a topic
  HATCHET.streams.reset("orders")

  HATCHET.streams.subscribe("orders") do |event|
    puts event.payload
  end
end

def reset_to_cursor
  # > Reset to a cursor
  cursor = nil

  HATCHET.streams.subscribe("orders") do |event|
    cursor = event.cursor
    break
  end

  HATCHET.streams.reset("orders", cursor: cursor)

  HATCHET.streams.subscribe("orders") do |event|
    puts event.payload
  end
end

if __FILE__ == $PROGRAM_NAME
  consume
  publish_from_anywhere
  namespaces
end
