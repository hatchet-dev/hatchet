# frozen_string_literal: true

def mock_classify(message)
  lower = message.downcase
  return 'support' if %w[bug error help].any? { |w| lower.include?(w) }
  return 'sales' if %w[price buy plan].any? { |w| lower.include?(w) }

  'other'
end

def mock_reply(message, role)
  case role
  when 'support'
    "[Support] I can help with that technical issue. Let me look into: #{message}"
  when 'sales'
    "[Sales] Great question about pricing! Here's what I can tell you about: #{message}"
  else
    "[General] Thanks for reaching out. Regarding: #{message}"
  end
end
