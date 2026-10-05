import { Code, ConnectError } from '@connectrpc/connect';
import { BulkTriggerIdempotencyCollisionErrorSchema } from '@hatchet/protoc-es/v1/workflows_pb';
import { BulkTriggerIdempotencyCollisionError } from '@util/errors/bulk-trigger-idempotency-collision-error';
import { IdempotencyCollisionError } from '@util/errors/idempotency-collision-error';
import { AdminClient } from './admin';

function makeBulkTriggerAlreadyExistsError(
  successfulIds: string[],
  collisions: { existingRunExternalId: string }[]
): Error {
  return new ConnectError('idempotency key collision', Code.AlreadyExists, undefined, [
    {
      desc: BulkTriggerIdempotencyCollisionErrorSchema,
      value: {
        successfulWorkflowRunExternalIds: successfulIds,
        collisions: collisions.map((c) => ({
          existingRunExternalId: c.existingRunExternalId,
          collidingRunExternalId: '',
        })),
      },
    },
  ]);
}

function createMockAdmin(namespace?: string): AdminClient {
  const admin = Object.create(AdminClient.prototype) as AdminClient;

  admin.config = {
    namespace,
    logger: () => ({ warn: jest.fn(), debug: jest.fn(), error: jest.fn(), info: jest.fn() }),
  } as any;
  admin.logger = admin.config.logger('test', undefined as any);
  admin.listenerClient = { get: jest.fn() } as any;
  admin.runs = {} as any;

  admin.workflowsGrpc = {
    triggerWorkflow: jest.fn().mockResolvedValue({ workflowRunId: 'run-1' }),
    bulkTriggerWorkflow: jest.fn().mockResolvedValue({ workflowRunIds: ['run-1'] }),
  } as any;

  return admin;
}

describe('AdminClient workflow name normalization', () => {
  it('runWorkflow lowercases PascalCase workflow name', async () => {
    const admin = createMockAdmin();
    await admin.runWorkflow('MyPascalWorkflow', { hello: 'world' });

    expect(admin.workflowsGrpc.triggerWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'mypascalworkflow' })
    );
  });

  it('runWorkflow lowercases camelCase workflow name', async () => {
    const admin = createMockAdmin();
    await admin.runWorkflow('concurrencyCancelNewest', {});

    expect(admin.workflowsGrpc.triggerWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'concurrencycancelnewest' })
    );
  });

  it('runWorkflow preserves already-lowercase name', async () => {
    const admin = createMockAdmin();
    await admin.runWorkflow('my-workflow', {});

    expect(admin.workflowsGrpc.triggerWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'my-workflow' })
    );
  });

  it('runWorkflow lowercases with namespace', async () => {
    const admin = createMockAdmin('ns-');
    await admin.runWorkflow('MyWorkflow', {});

    expect(admin.workflowsGrpc.triggerWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'ns-myworkflow' })
    );
  });

  it('runWorkflows lowercases workflow names in batch', async () => {
    const admin = createMockAdmin();
    await admin.runWorkflows([
      { workflowName: 'WorkflowOne', input: {} },
      { workflowName: 'WorkflowTwo', input: {} },
    ]);

    expect(admin.workflowsGrpc.bulkTriggerWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({
        workflows: expect.arrayContaining([
          expect.objectContaining({ name: 'workflowone' }),
          expect.objectContaining({ name: 'workflowtwo' }),
        ]),
      })
    );
  });
});

describe('AdminClient bulk trigger idempotency collision', () => {
  it('throws BulkTriggerIdempotencyCollisionError on ALREADY_EXISTS', async () => {
    const admin = createMockAdmin();
    const bulkError = makeBulkTriggerAlreadyExistsError(
      ['run-success-1'],
      [{ existingRunExternalId: 'run-existing-1' }]
    );

    admin.workflowsGrpc = {
      bulkTriggerWorkflow: jest.fn().mockRejectedValue(bulkError),
    } as any;

    await expect(
      admin.runWorkflows([
        { workflowName: 'my-workflow', input: {} },
        { workflowName: 'my-workflow', input: {} },
      ])
    ).rejects.toBeInstanceOf(BulkTriggerIdempotencyCollisionError);
  });

  it('exposes successful IDs and individual collision errors', async () => {
    const admin = createMockAdmin();
    const bulkError = makeBulkTriggerAlreadyExistsError(
      ['run-success-1', 'run-success-2'],
      [{ existingRunExternalId: 'run-existing-1' }]
    );

    admin.workflowsGrpc = {
      bulkTriggerWorkflow: jest.fn().mockRejectedValue(bulkError),
    } as any;

    let caught: BulkTriggerIdempotencyCollisionError | undefined;
    try {
      await admin.runWorkflows([
        { workflowName: 'my-workflow', input: {} },
        { workflowName: 'my-workflow', input: {} },
        { workflowName: 'my-workflow', input: {} },
      ]);
    } catch (e) {
      caught = e as BulkTriggerIdempotencyCollisionError;
    }

    expect(caught).toBeInstanceOf(BulkTriggerIdempotencyCollisionError);
    expect(caught?.successfulWorkflowRunExternalIds).toEqual(['run-success-1', 'run-success-2']);
    expect(caught?.collisions).toHaveLength(1);
    expect(caught?.collisions[0]).toBeInstanceOf(IdempotencyCollisionError);
    expect(caught?.collisions[0].existingRunExternalId).toBe('run-existing-1');
  });
});
