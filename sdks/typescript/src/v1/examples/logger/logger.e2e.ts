import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import { loggingWorkflow } from './workflow';

describe('logger-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'logger-e2e-worker',
      workflows: [loggingWorkflow],
      slots: 10,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it('runs logging workflow tasks', async () => {
    const result = await loggingWorkflow.run({});

    // python asserts only root_logger, but we validate both tasks here
    expect((result as any).root_logger.status).toBe('success');
    expect((result as any).context_logger.status).toBe('success');
  }, 90_000);
});
