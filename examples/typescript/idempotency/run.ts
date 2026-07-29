import { IdempotencyCollisionError } from '@hatchet-dev/typescript-sdk/v1';
import { idempotentTask } from './workflow';

async function main() {
  // > trigger
  const ref1 = await idempotentTask.runNoWait({ id: '123' });

  let runId2: string;
  try {
    const ref2 = await idempotentTask.runNoWait({ id: '123' });
    runId2 = await ref2.getWorkflowRunId();
  } catch (e) {
    if (e instanceof IdempotencyCollisionError) {
      console.log(
        `Run with external ID ${e.existingRunExternalId} already exists for this idempotency key`
      );
      runId2 = e.existingRunExternalId;
    } else {
      throw e;
    }
  }

  const res1 = await ref1.result();
  console.log(`Result: ${JSON.stringify(res1)}, run ID: ${runId2}`);
}

main().catch(console.error);
