# frozen_string_literal: true

# Mock LLM - no external API dependencies
def generate(prompt)
  { 'content' => "Generated for: #{prompt[0, 50]}...", 'valid' => true }
end
