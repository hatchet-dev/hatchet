import { hatchet } from '../../hatchet-client';
import { embed } from './mock-embedding';

// > Step 01 Define Workflow
type DocInput = { doc_id: string; content: string };

const ragWf = hatchet.workflow<DocInput>({ name: 'RAGPipeline' });

// > Step 02 Define Ingest Task
const ingest = ragWf.task({
  name: 'ingest',
  fn: async (input) => ({ doc_id: input.doc_id, content: input.content }),
});


// > Step 03 Chunk Task
function chunkContent(content: string, chunkSize = 100): string[] {
  const chunks: string[] = [];
  for (let i = 0; i < content.length; i += chunkSize) {
    chunks.push(content.slice(i, i + chunkSize));
  }
  return chunks;
}

// > Step 04 Embed Task
const embedChunkTask = hatchet.task<{ chunk: string }>({
  name: 'embed-chunk',
  fn: async (input) => ({ vector: embed(input.chunk) }),
});

const chunkAndEmbed = ragWf.durableTask({
  name: 'chunk-and-embed',
  parents: [ingest],
  fn: async (input, ctx) => {
    const ingested = await ctx.parentOutput(ingest);
    const chunks: string[] = [];
    for (let i = 0; i < ingested.content.length; i += 100) {
      chunks.push(ingested.content.slice(i, i + 100));
    }
    const results = await Promise.all(chunks.map((chunk) => embedChunkTask.run({ chunk })));
    return { doc_id: ingested.doc_id, vectors: results.map((r) => r.vector) };
  },
});


// > Step 05 Query Task
type QueryInput = { query: string; top_k?: number };

const queryTask = hatchet.durableTask<QueryInput>({
  name: 'rag-query',
  fn: async (input) => {
    const { vector } = await embedChunkTask.run({ chunk: input.query });
    // Replace with a real vector DB lookup in production
    return { query: input.query, vector, results: [] };
  },
});

export { ragWf, embedChunkTask, queryTask };
