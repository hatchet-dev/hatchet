# frozen_string_literal: true

# Third-party integration - requires: bundle add openai
# See: /guides/rag-and-indexing

require "openai"

OpenAI::Client.new

# > OpenAI embedding usage
def embed(text)
  response = client.embeddings(parameters: { model: "text-embedding-3-small", input: text })
  response.dig("data", 0, "embedding") || []
end
# !!
