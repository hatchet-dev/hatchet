// Third-party integration - requires: pnpm add openai
// See: /guides/rag-and-indexing

import OpenAI from 'openai';

const client = new OpenAI();

// > OpenAI embedding usage
export async function embed(text: string): Promise<number[]> {
  const r = await client.embeddings.create({
    model: 'text-embedding-3-small',
    input: text,
  });
  return r.data[0]?.embedding ?? [];
}
// !!
