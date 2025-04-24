/* eslint-disable no-console */
// ❓ Running a Task with Results
import { cancellation } from './workflow';
// ...
async function main() {
  // 👀 Run the workflow with results
  const res = await cancellation.run({});

  // 👀 Access the results of the workflow
  console.log(res.Completed);
  // !!
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
