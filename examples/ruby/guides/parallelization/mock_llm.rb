# frozen_string_literal: true

def mock_generate_content(message)
  "Here is a helpful response to: #{message}"
end

def mock_safety_check(message)
  if message.downcase.include?('unsafe')
    { 'safe' => false, 'reason' => 'Content flagged as potentially unsafe.' }
  else
    { 'safe' => true, 'reason' => 'Content is appropriate.' }
  end
end

def mock_evaluate_content(content)
  score = content.length > 20 ? 0.85 : 0.3
  { 'score' => score, 'approved' => score >= 0.7 }
end
