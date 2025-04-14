/* eslint-disable no-console */
import { withMiddleware } from './workflow';

async function main() {
  // ❓ Running a Task
  const res = await withMiddleware.run({
    Message: 'HeLlO WoRlD',
  });

  // 👀 Access the results of the Task
  console.log(res.TransformedMessage);
  // !!
}

if (require.main === module) {
  main();
}
