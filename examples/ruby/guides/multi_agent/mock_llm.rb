# frozen_string_literal: true

@orchestrator_call_count = 0

def mock_orchestrator_llm(messages)
  @orchestrator_call_count += 1
  case @orchestrator_call_count
  when 1
    { "done" => false, "content" => "", "tool_call" => { "name" => "research", "args" => { "task" => "Find key facts about the topic" } } }
  when 2
    { "done" => false, "content" => "", "tool_call" => { "name" => "writing", "args" => { "task" => "Write a summary from the research" } } }
  else
    { "done" => true, "content" => "Here is the final report combining research and writing." }
  end
end

def mock_specialist_llm(task, role)
  "[#{role}] Completed: #{task}"
end
