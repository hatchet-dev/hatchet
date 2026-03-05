// Third-party integration - requires: pnpm add @mendable/firecrawl-js
// See: /guides/web-scraping

import FirecrawlApp from '@mendable/firecrawl-js';

const firecrawl = new FirecrawlApp({ apiKey: process.env.FIRECRAWL_API_KEY! });

// > Firecrawl usage
export async function scrapeUrl(url: string) {
  const result = await firecrawl.scrapeUrl(url, { formats: ['markdown'] });
  return {
    url,
    content: result.markdown,
    metadata: result.metadata,
  };
}
