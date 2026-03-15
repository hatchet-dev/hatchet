// Third-party integration - requires: pnpm add openai
// See: /guides/web-scraping

import OpenAI from 'openai';

const openai = new OpenAI();

// > OpenAI web search usage
export async function searchAndExtract(query: string) {
  const response = await openai.responses.create({
    model: 'gpt-4o-mini',
    tools: [{ type: 'web_search' }],
    input: query,
  });
  return { query, content: response.output_text };
}
