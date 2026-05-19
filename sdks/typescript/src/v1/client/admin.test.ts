import { AdminClient } from './admin';

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
