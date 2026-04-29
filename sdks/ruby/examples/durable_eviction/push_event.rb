# frozen_string_literal: true

require "json"

require_relative "worker"

def parse_payload(argv)
  return({ "ok" => true }) if argv.empty?

  JSON.parse(argv.first)
rescue JSON::ParserError => e
  warn "Invalid JSON payload: #{e.message}"
  warn %(Usage: bundle exec ruby push_event.rb '{"ok":true}')
  exit 1
end

def main
  payload = parse_payload(ARGV)
  response = HATCHET.events.push(EVENT_KEY, payload)

  puts "Pushed event #{EVENT_KEY}"
  puts response.inspect
end

main if __FILE__ == $PROGRAM_NAME
