import { hatchet } from '../../hatchet-client';
import { embed } from './mock-embedding';

type DocInput = { doc_id: string; content: string };

// > Step 01 Define Ingest Task
const ragWf = hatchet.workflow<DocInput>({ name: 'RAGPipeline' });

const ingest = ragWf.task({
  name: 'ingest',
  fn: async (input) => ({ doc_id: input.doc_id, content: input.content }),
});


// > Step 02 Chunk Task
function chunkContent(content: string, chunkSize = 100): string[] {
  const chunks: string[] = [];
  for (let i = 0; i < content.length; i += chunkSize) {
    chunks.push(content.slice(i, i + chunkSize));
  }
  return chunks;
}

// > Step 03 Embed Task
const chunkAndEmbed = ragWf.task({
  name: 'chunk-and-embed',
  parents: [ingest],
  fn: async (input, ctx) => {
    const ingested = await ctx.parentOutput(ingest);
    const chunks: string[] = [];
    for (let i = 0; i < ingested.content.length; i += 100) {
      chunks.push(ingested.content.slice(i, i + 100));
    }
    const vectors = chunks.map((c) => embed(c));
    return { doc_id: ingested.doc_id, vectors };
  },
});


export { ragWf };
