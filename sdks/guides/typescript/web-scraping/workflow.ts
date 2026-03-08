import { hatchet } from '../../hatchet-client';
import { mockScrape } from './mock-scraper';

type ScrapeInput = { url: string };

// > Step 01 Define Scrape Task
const scrapeTask = hatchet.task({
  name: 'scrape-url',
  executionTimeout: '2m',
  retries: 2,
  fn: async (input: ScrapeInput) => {
    return mockScrape(input.url);
  },
});
// !!

// > Step 02 Process Content
const processTask = hatchet.task({
  name: 'process-content',
  fn: async (input: { url: string; content: string }) => {
    const links = [...input.content.matchAll(/https?:\/\/[^\s<>"']+/g)].map((m) => m[0]);
    const summary = input.content.slice(0, 200).trim();
    const wordCount = input.content.split(/\s+/).filter(Boolean).length;
    return { summary, wordCount, links };
  },
});
// !!

// > Step 03 Cron Workflow
const scrapeWorkflow = hatchet.workflow({
  name: 'WebScrapeWorkflow',
  on: { cron: '0 */6 * * *' },
});

scrapeWorkflow.task({
  name: 'scheduled-scrape',
  fn: async () => {
    const urls = [
      'https://example.com/pricing',
      'https://example.com/blog',
      'https://example.com/docs',
    ];

    const results = [];
    for (const url of urls) {
      const scraped = await scrapeTask.run({ url });
      const processed = await processTask.run({ url, content: scraped.content });
      results.push({ url, ...processed });
    }
    return { refreshed: results.length, results };
  },
});
// !!

// > Step 04 Rate Limited Scrape
const SCRAPE_RATE_LIMIT_KEY = 'scrape-rate-limit';

const rateLimitedScrapeTask = hatchet.task({
  name: 'rate-limited-scrape',
  executionTimeout: '2m',
  retries: 2,
  rateLimits: [
    {
      staticKey: SCRAPE_RATE_LIMIT_KEY,
      units: 1,
    },
  ],
  fn: async (input: ScrapeInput) => {
    return mockScrape(input.url);
  },
});
// !!

export { scrapeTask, processTask, scrapeWorkflow, rateLimitedScrapeTask, SCRAPE_RATE_LIMIT_KEY };
