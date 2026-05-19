# frozen_string_literal: true

require 'hatchet-sdk'
require_relative 'mock_embedding'

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Define Workflow
RAG_WF = HATCHET.workflow(name: 'RAGPipeline')

# > Step 02 Define Ingest Task
INGEST = RAG_WF.task(:ingest) do |input, _ctx|
  { 'doc_id' => input['doc_id'], 'content' => input['content'] }
end


# > Step 03 Chunk Task
def chunk_content(content, chunk_size = 100)
  content.scan(/.{1,#{chunk_size}}/)
end

# > Step 04 Embed Task
EMBED_CHUNK_TASK = HATCHET.task(name: 'embed-chunk') do |input, _ctx|
  { 'vector' => embed(input['chunk']) }
end

RAG_WF.durable_task(:chunk_and_embed, parents: [INGEST]) do |_input, ctx|
  ingested = ctx.task_output(INGEST)
  content = ingested['content']
  chunks = content.scan(/.{1,100}/)
  results = EMBED_CHUNK_TASK.run_many(
    chunks.map { |c| EMBED_CHUNK_TASK.create_bulk_run_item(input: { 'chunk' => c }) }
  )
  { 'doc_id' => ingested['doc_id'], 'vectors' => results.map { |r| r['vector'] } }
end


# > Step 05 Query Task
QUERY_TASK = HATCHET.durable_task(name: 'rag-query') do |input, _ctx|
  result = EMBED_CHUNK_TASK.run('chunk' => input['query'])
  # Replace with a real vector DB lookup in production
  { 'query' => input['query'], 'vector' => result['vector'], 'results' => [] }
end

def main
  # > Step 06 Run Worker
  worker = HATCHET.worker('rag-worker', workflows: [RAG_WF, EMBED_CHUNK_TASK, QUERY_TASK])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
