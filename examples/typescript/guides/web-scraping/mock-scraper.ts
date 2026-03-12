export interface ScrapeResult {
  url: string;
  title: string;
  content: string;
  scrapedAt: string;
}

export function mockScrape(url: string): ScrapeResult {
  return {
    url,
    title: `Page: ${url}`,
    content: `Mock scraped content from ${url}. In production, use Firecrawl, Browserbase, or Playwright here.`,
    scrapedAt: new Date().toISOString(),
  };
}

export function mockExtract(content: string): Record<string, string> {
  return {
    summary: content.slice(0, 80),
    wordCount: String(content.split(' ').length),
  };
}
