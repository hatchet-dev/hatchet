# frozen_string_literal: true

require "hatchet-sdk"
require_relative "mock_classifier"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Classify Task
CLASSIFY_TASK = HATCHET.durable_task(name: "ClassifyMessage") do |input, _ctx|
  { "category" => mock_classify(input["message"]) }
end

# > Step 02 Specialist Tasks
SUPPORT_TASK = HATCHET.durable_task(name: "HandleSupport") do |input, _ctx|
  { "response" => mock_reply(input["message"], "support"), "category" => "support" }
end

SALES_TASK = HATCHET.durable_task(name: "HandleSales") do |input, _ctx|
  { "response" => mock_reply(input["message"], "sales"), "category" => "sales" }
end

DEFAULT_TASK = HATCHET.durable_task(name: "HandleDefault") do |input, _ctx|
  { "response" => mock_reply(input["message"], "other"), "category" => "other" }
end

# > Step 03 Router Task
ROUTER_TASK = HATCHET.durable_task(name: "MessageRouter", execution_timeout: "2m") do |input, _ctx|
  classification = CLASSIFY_TASK.run("message" => input["message"])

  case classification["category"]
  when "support"
    SUPPORT_TASK.run("message" => input["message"])
  when "sales"
    SALES_TASK.run("message" => input["message"])
  else
    DEFAULT_TASK.run("message" => input["message"])
  end
end

def main
  # > Step 04 Run Worker
  worker = HATCHET.worker("routing-worker", slots: 5,
                                            workflows: [CLASSIFY_TASK, SUPPORT_TASK, SALES_TASK, DEFAULT_TASK, ROUTER_TASK],)
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
