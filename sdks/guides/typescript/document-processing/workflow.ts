import { hatchet } from '../../hatchet-client';
import { parseDocument } from './mock-ocr';

type DocInput = { doc_id: string; content: Uint8Array };

// > Step 01 Define DAG
const docWf = hatchet.workflow<DocInput>({ name: 'DocumentPipeline' });

const ingest = docWf.task({
  name: 'ingest',
  fn: async (input) => ({ doc_id: input.doc_id, content: input.content }),
});

// !!

// > Step 02 Parse Stage
const parse = docWf.task({
  name: 'parse',
  parents: [ingest],
  fn: async (input, ctx) => {
    const ingested = await ctx.parentOutput(ingest);
    const text = parseDocument(ingested.content);
    return { doc_id: input.doc_id, text };
  },
});

// !!

// > Step 03 Extract Stage
const extract = docWf.task({
  name: 'extract',
  parents: [parse],
  fn: async (input, ctx) => {
    const parsed = await ctx.parentOutput(parse);
    return { doc_id: parsed.doc_id, entities: ['entity1', 'entity2'] };
  },
});

// !!

export { docWf };
