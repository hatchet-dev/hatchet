# frozen_string_literal: true

# Mock embedding - no external API dependencies
def embed(_text)
  [0.1] * 64
end
