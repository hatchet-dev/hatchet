# frozen_string_literal: true

require 'hatchet-sdk'
require_relative 'mock_classifier'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

CLASSIFY_WF = HATCHET.workflow(name: 'ClassifyMessage')
SUPPORT_WF = HATCHET.workflow(name: 'HandleSupport')
SALES_WF = HATCHET.workflow(name: 'HandleSales')
DEFAULT_WF = HATCHET.workflow(name: 'HandleDefault')

# > Step 01 Classify Task
CLASSIFY_WF.task(:classify_message) do |input, _ctx|
  { 'category' => mock_classify(input['message']) }
end
# !!

# > Step 02 Specialist Tasks
SUPPORT_WF.task(:handle_support) do |input, _ctx|
  { 'response' => mock_reply(input['message'], 'support'), 'category' => 'support' }
end

SALES_WF.task(:handle_sales) do |input, _ctx|
  { 'response' => mock_reply(input['message'], 'sales'), 'category' => 'sales' }
end

DEFAULT_WF.task(:handle_default) do |input, _ctx|
  { 'response' => mock_reply(input['message'], 'other'), 'category' => 'other' }
end
# !!

# > Step 03 Router Task
ROUTER_TASK = HATCHET.durable_task(name: 'MessageRouter', execution_timeout: '2m') do |input, _ctx|
  classification = CLASSIFY_WF.run('message' => input['message'])

  case classification['category']
  when 'support'
    SUPPORT_WF.run('message' => input['message'])
  when 'sales'
    SALES_WF.run('message' => input['message'])
  else
    DEFAULT_WF.run('message' => input['message'])
  end
end
# !!

def main
  # > Step 04 Run Worker
  worker = HATCHET.worker('routing-worker', slots: 5,
                                            workflows: [CLASSIFY_WF, SUPPORT_WF, SALES_WF, DEFAULT_WF, ROUTER_TASK])
  worker.start
  # !!
end

main if __FILE__ == $PROGRAM_NAME
