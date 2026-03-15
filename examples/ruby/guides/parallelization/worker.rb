# frozen_string_literal: true

require 'hatchet-sdk'
require_relative 'mock_llm'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

CONTENT_WF = HATCHET.workflow(name: 'GenerateContent')
SAFETY_WF = HATCHET.workflow(name: 'SafetyCheck')
EVALUATOR_WF = HATCHET.workflow(name: 'EvaluateContent')

# > Step 01 Parallel Tasks
CONTENT_WF.task(:generate_content) do |input, _ctx|
  { 'content' => mock_generate_content(input['message']) }
end

SAFETY_WF.task(:safety_check) do |input, _ctx|
  mock_safety_check(input['message'])
end

EVALUATOR_WF.task(:evaluate_content) do |input, _ctx|
  mock_evaluate_content(input['content'])
end

# > Step 02 Sectioning
SECTIONING_TASK = HATCHET.durable_task(name: 'ParallelSectioning', execution_timeout: '2m') do |input, _ctx|
  threads = []
  threads << Thread.new { CONTENT_WF.run('message' => input['message']) }
  threads << Thread.new { SAFETY_WF.run('message' => input['message']) }
  content_result, safety_result = threads.map(&:value)

  if safety_result['safe']
    { 'blocked' => false, 'content' => content_result['content'] }
  else
    { 'blocked' => true, 'reason' => safety_result['reason'] }
  end
end

# > Step 03 Voting
VOTING_TASK = HATCHET.durable_task(name: 'ParallelVoting', execution_timeout: '3m') do |input, _ctx|
  threads = 3.times.map { Thread.new { EVALUATOR_WF.run('content' => input['content']) } }
  votes = threads.map(&:value)

  approvals = votes.count { |v| v['approved'] }
  avg_score = votes.sum { |v| v['score'] } / votes.size.to_f

  { 'approved' => approvals >= 2, 'average_score' => avg_score, 'votes' => votes.size }
end

def main
  # > Step 04 Run Worker
  worker = HATCHET.worker('parallelization-worker', slots: 10,
                                                    workflows: [CONTENT_WF, SAFETY_WF, EVALUATOR_WF, SECTIONING_TASK, VOTING_TASK])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
