

import { hatchet } from '../hatchet-client';
import { SimpleOutput } from './stub-workflow';
// ❓ Enqueuing a Workflow (Fire and Forget)
import { simple } from './workflow';
// ...

async function main() {
  // 👀 Enqueue the workflow
  const run = await simple.runNoWait({
    Message: 'hello',
  });

  // 👀 Get the run ID of the workflow
  const runId = await run.getWorkflowRunId();
  // It may be helpful to store the run ID of the workflow
  // in a database or other persistent storage for later use
  console.log(runId);

  // ❓ Subscribing to results
  // the return object of the enqueue method is a WorkflowRunRef which includes a listener for the result of the workflow
  const result = await run.result();
  console.log(result);

  // if you need to subscribe to the result of the workflow at a later time, you can use the runRef method and the stored runId
  const ref = hatchet.runRef<SimpleOutput>(runId);
  const result2 = await ref.result();
  console.log(result2);

}

if (require.main === module) {
  main();
}
