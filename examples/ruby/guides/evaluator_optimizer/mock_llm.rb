# frozen_string_literal: true

@generate_count = 0

def mock_generate(prompt)
  @generate_count += 1
  if @generate_count == 1
    "Check out our product! Buy now!"
  else
    "Discover how our tool saves teams 10 hours/week. Try it free."
  end
end

def mock_evaluate(draft)
  if draft.length < 40
    { "score" => 0.4, "feedback" => "Too short and pushy. Add a specific benefit and soften the CTA." }
  else
    { "score" => 0.9, "feedback" => "Clear value prop, appropriate tone." }
  end
end
