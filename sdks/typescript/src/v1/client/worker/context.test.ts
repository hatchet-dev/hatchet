import { createAction } from '@hatchet/clients/dispatcher/action-listener';
import { ActionType } from '@hatchet-dev/typescript-sdk/protoc/dispatcher';
import type { HatchetClient } from '@hatchet/v1';
import { Context } from './context';
import type { InternalWorker } from './worker-internal';

describe('Context', () => {
  it('returns the workflow name separately from the task name', () => {
    const action = createAction({
      tenantId: 'tenant-id',
      workflowRunId: 'workflow-run-id',
      getGroupKeyRunId: '',
      jobId: 'task-id',
      jobName: 'my-task',
      jobRunId: 'task-run-id',
      taskId: 'task-id',
      taskRunExternalId: 'task-run-id',
      actionId: 'my-workflow:my-task',
      actionType: ActionType.START_STEP_RUN,
      actionPayload: JSON.stringify({ input: {} }),
      taskName: 'my-task',
      retryCount: 0,
      priority: 1,
    });
    const logger = {
      error: jest.fn(),
    };
    const client = {
      config: {
        logger: () => logger,
        log_level: 'INFO',
      },
    } as unknown as HatchetClient;
    const worker = {} as InternalWorker;

    const context = new Context(action, client, worker);

    expect(context.workflowName()).toBe('my-workflow');
    expect(context.taskName()).toBe('my-task');
  });
});
