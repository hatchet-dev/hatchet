/* eslint-disable no-console */
import { parent } from './workflow-with-child';

async function main() {
  // ❓ Running a Task
  const res = await parent.run({
    Message: 'HeLlO WoRlD',
  });

  // 👀 Access the results of the Task
  console.log(res.TransformedMessage);
  // !!
}

if (require.main === module) {
  main();
}
