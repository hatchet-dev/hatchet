import { hatchetWithMiddleware } from './client';
import { taskWithMiddleware } from './workflow';

async function main() {
  const worker = await hatchetWithMiddleware.worker('task-with-middleware', {
    workflows: [taskWithMiddleware],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
