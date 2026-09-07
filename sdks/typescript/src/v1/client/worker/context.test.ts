import { createAction } from '@hatchet/clients/dispatcher/action-listener';
import { ActionType } from '@hatchet-dev/typescript-sdk/protoc/dispatcher';
import type { HatchetClient } from '@hatchet/v1';
import { Context, DurableContext } from './context';
import type { InternalWorker } from './worker-internal';
import type { DurableListenerClient } from '@hatchet/clients/listeners/durable-listener/durable-listener-client';

describe('Context', () => {
  it('returns the workflow name without changing the deprecated method', () => {
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

    expect(context.workflowName()).toBe('my-task');
    expect(context.workflowNameV1()).toBe('my-workflow');
    expect(context.taskName()).toBe('my-task');
  });
});

describe('DurableContext', () => {
  function buildDurableContext() {
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
        namespace: '',
      },
    } as unknown as HatchetClient;
    const worker = {} as InternalWorker;
    const durableListener = {} as DurableListenerClient;

    // The eviction-enabled durable trigger-opts path is only reachable once the
    // engine version supports eviction (see `supportsEviction`).
    return new DurableContext(action, client, worker, durableListener, undefined, 'v0.105.16');
  }

  // Regression test for https://github.com/hatchet-dev/hatchet/issues/4904:
  // DurableContext's eviction-enabled trigger-opts builder hard-coded
  // `desiredWorkerLabels: {}` instead of converting the caller-supplied labels,
  // so labels passed to `spawnChild`/`spawnChildren` were silently dropped.
  it('preserves desiredWorkerLabels when building durable child trigger opts', () => {
    const context = buildDurableContext();

    const desiredWorkerLabels = {
      color: { value: 'blue', required: true, comparator: 0 },
    };

    const { triggerOpts } = (context as any)._buildTriggerOpts(
      'child-workflow',
      { foo: 'bar' },
      {
        key: 'child-key',
        desiredWorkerLabels,
      }
    );

    expect(triggerOpts.desiredWorkerLabels).toEqual({
      color: {
        strValue: 'blue',
        intValue: undefined,
        required: true,
        weight: undefined,
        comparator: 0,
      },
    });

    // Other fields must remain correctly populated (unchanged by this fix).
    expect(triggerOpts.parentId).toBe('workflow-run-id');
    expect(triggerOpts.parentTaskRunExternalId).toBe('task-run-id');
    expect(triggerOpts.childIndex).toBeDefined();
    expect(triggerOpts.childKey).toBe('child-key');
  });

  it('produces empty desiredWorkerLabels when the caller does not supply labels', () => {
    const context = buildDurableContext();

    const { triggerOpts } = (context as any)._buildTriggerOpts(
      'child-workflow',
      { foo: 'bar' },
      {
        key: 'child-key',
      }
    );

    expect(triggerOpts.desiredWorkerLabels).toEqual({});
  });

  it('preserves desiredWorkerLabels for each child when building bulk durable trigger opts', () => {
    const context = buildDurableContext();

    const blueLabels = { color: { value: 'blue', required: true, comparator: 0 } };
    const redLabels = { color: { value: 'red', required: true, comparator: 0 } };

    const first = (context as any)._buildTriggerOpts(
      'child-workflow-1',
      { foo: 'bar' },
      {
        key: 'child-1',
        desiredWorkerLabels: blueLabels,
      }
    );
    const second = (context as any)._buildTriggerOpts(
      'child-workflow-2',
      { foo: 'baz' },
      {
        key: 'child-2',
        desiredWorkerLabels: redLabels,
      }
    );

    expect(first.triggerOpts.desiredWorkerLabels).toEqual({
      color: {
        strValue: 'blue',
        intValue: undefined,
        required: true,
        weight: undefined,
        comparator: 0,
      },
    });
    expect(second.triggerOpts.desiredWorkerLabels).toEqual({
      color: {
        strValue: 'red',
        intValue: undefined,
        required: true,
        weight: undefined,
        comparator: 0,
      },
    });
  });
});
