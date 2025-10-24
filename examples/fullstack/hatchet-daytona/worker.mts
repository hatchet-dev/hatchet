import { hatchet } from './clients';
import { orchestrate } from './orchestration';
import { generate, execute } from './tasks';

async function main() {
  const worker = await hatchet.worker('simple-worker', {
    workflows: [generate, execute, orchestrate],
  });

  await worker.start();
}

// Run main function when this file is executed directly
main().catch(console.error);
