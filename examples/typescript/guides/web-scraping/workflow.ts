import { hatchet } from '../../hatchet-client';
import { mockScrape, mockExtract } from './mock-scraper';

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

// > Step 02 Process Content
const processTask = hatchet.task({
  name: 'process-content',
  fn: async (input: { url: string; content: string }) => {
    return mockExtract(input.content);
  },
});

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

export { scrapeTask, processTask, scrapeWorkflow };
