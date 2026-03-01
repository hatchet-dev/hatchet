# frozen_string_literal: true

# Mock LLM and tools - no external API dependencies
@llm_call_count = 0

def call_llm(messages)
  @llm_call_count += 1
  if @llm_call_count == 1
    { "content" => "", "tool_calls" => [{ "name" => "get_weather", "args" => { "location" => "SF" } }], "done" => false }
  else
    { "content" => "It's 72°F and sunny in SF.", "tool_calls" => [], "done" => true }
  end
end

def run_tool(name, args)
  if name == "get_weather"
    loc = args["location"] || "unknown"
    "Weather in #{loc}: 72°F, sunny"
  else
    "Unknown tool: #{name}"
  end
end
