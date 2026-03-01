import { RateLimitDuration } from '@hatchet-dev/typescript-sdk/protoc/v1/workflows';
import { hatchet } from '../../hatchet-client';
import { scrapeTask, processTask, scrapeWorkflow, rateLimitedScrapeTask, SCRAPE_RATE_LIMIT_KEY } from './workflow';

async function main() {
  // > Step 05 Run Worker
  await hatchet.ratelimits.upsert({
    key: SCRAPE_RATE_LIMIT_KEY,
    limit: 10,
    duration: RateLimitDuration.MINUTE,
  });

  const worker = await hatchet.worker('web-scraping-worker', {
    workflows: [scrapeTask, processTask, scrapeWorkflow, rateLimitedScrapeTask],
    slots: 5,
  });
  await worker.start();
}

if (require.main === module) {
  main();
}
