import { makeE2EClient } from '../__e2e__/harness';
import { createLoggingWorkflow } from './workflow';

describe('logger-e2e', () => {
  const hatchet = makeE2EClient();
  const loggingWorkflow = createLoggingWorkflow(hatchet);

  it('runs logging workflow tasks', async () => {
    const result = await loggingWorkflow.run({});

    // python asserts only root_logger, but we validate both tasks here
    expect((result as any).root_logger.status).toBe('success');
    expect((result as any).context_logger.status).toBe('success');
  }, 90_000);
});
