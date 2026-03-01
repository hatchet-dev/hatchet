# frozen_string_literal: true

require 'hatchet-sdk'
require_relative 'mock_embedding'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Define Ingest Task
RAG_WF = HATCHET.workflow(name: 'RAGPipeline')

INGEST = RAG_WF.task(:ingest) do |input, _ctx|
  { 'doc_id' => input['doc_id'], 'content' => input['content'] }
end


# > Step 02 Chunk Task
def chunk_content(content, chunk_size = 100)
  content.scan(/.{1,#{chunk_size}}/)
end

# > Step 03 Embed Task
RAG_WF.task(:chunk_and_embed, parents: [INGEST]) do |_input, ctx|
  ingested = ctx.task_output(INGEST)
  content = ingested['content']
  chunks = content.scan(/.{1,100}/)
  vectors = chunks.map { |c| embed(c) }
  { 'doc_id' => ingested['doc_id'], 'vectors' => vectors }
end


def main
  # > Step 04 Run Worker
  worker = HATCHET.worker('rag-worker', workflows: [RAG_WF])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
