// Third-party integration - requires: pnpm add cohere-ai
// See: /guides/rag-and-indexing

import Cohere from 'cohere-ai';

const client = new Cohere();

// > Cohere embedding usage
export async function embed(text: string): Promise<number[]> {
  const r = await client.embed({
    texts: [text],
    model: 'embed-english-v3.0',
    inputType: 'search_document',
  });
  return r.embeddings[0] ?? [];
}
