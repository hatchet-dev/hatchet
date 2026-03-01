import { hatchet } from '../../hatchet-client';
import { scrapeTask, processTask, scrapeWorkflow } from './workflow';

async function main() {
  // > Step 04 Run Worker
  const worker = await hatchet.worker('web-scraping-worker', {
    workflows: [scrapeTask, processTask, scrapeWorkflow],
    slots: 5,
  });
  await worker.start();
  // !!
}

if (require.main === module) {
  main();
}
