# frozen_string_literal: true

require "hatchet-sdk"
require_relative "mock_ocr"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Step 01 Define DAG
DOC_WF = HATCHET.workflow(name: "DocumentPipeline")

INGEST = DOC_WF.task(:ingest) do |input, ctx|
  { "doc_id" => input["doc_id"], "content" => input["content"] }
end


# > Step 02 Parse Stage
PARSE = DOC_WF.task(:parse, parents: [INGEST]) do |input, ctx|
  ingested = ctx.task_output(INGEST)
  text = parse_document(ingested["content"])
  { "doc_id" => input["doc_id"], "text" => text }
end


# > Step 03 Extract Stage
DOC_WF.task(:extract, parents: [PARSE]) do |input, ctx|
  parsed = ctx.task_output(PARSE)
  { "doc_id" => parsed["doc_id"], "entities" => %w[entity1 entity2] }
end


def main
  # > Step 04 Run Worker
  worker = HATCHET.worker("document-worker", workflows: [DOC_WF])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
