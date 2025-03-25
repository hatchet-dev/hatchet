/* eslint-disable no-console */
// ❓ Running a Workflow with Results
import { simple } from './workflow';
// ...
async function main() {
  // 👀 Run the workflow with results
  const res = await simple.run({
    Message: 'hello',
  });

  // 👀 Access the results of the workflow
  console.log(res['to-lower'].TransformedMessage);
  // !!
}

if (require.main === module) {
  main();
}
