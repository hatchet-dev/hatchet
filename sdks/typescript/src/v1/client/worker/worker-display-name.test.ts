import { InternalWorker } from './worker-internal';
import { WorkflowDeclaration, TaskWorkflowDeclaration } from '../../declaration';

// Build an InternalWorker whose admin.putWorkflow is a spy, so we can assert on
// the CreateWorkflowVersionRequest produced at registration time.
function createWorkerWithSpy() {
  const putWorkflow = jest.fn().mockResolvedValue(undefined);
  const client = {
    config: {
      namespace: '',
      logger: () => new Proxy({}, { get: () => jest.fn() }),
      log_level: 'OFF',
      healthcheck: { enabled: false },
    },
    admin: { putWorkflow },
  } as any;

  const worker = new InternalWorker(client, { name: 'test-worker', handleKill: false });
  return { worker, putWorkflow };
}

describe('registerWorkflow display name', () => {
  it('threads a workflow-level displayName into CreateWorkflowVersionRequest', async () => {
    const { worker, putWorkflow } = createWorkerWithSpy();
    const wf = new WorkflowDeclaration({ name: 'dn-wf', displayName: 'input.customerName' });
    wf.task({ name: 'step', fn: async () => undefined });

    await worker.registerWorkflow(wf);

    expect(putWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({ displayName: 'input.customerName' })
    );
  });

  it('threads a task-level displayName into the CreateTaskOpts', async () => {
    const { worker, putWorkflow } = createWorkerWithSpy();
    const wf = new WorkflowDeclaration({ name: 'dn-task-wf' });
    wf.task({ name: 'step', displayName: "'enrich-' + input.name", fn: async () => undefined });

    await worker.registerWorkflow(wf);

    const [[req]] = putWorkflow.mock.calls;
    expect(req.tasks).toHaveLength(1);
    expect(req.tasks[0].displayName).toBe("'enrich-' + input.name");
  });

  it('leaves displayName unset on both surfaces when not provided', async () => {
    const { worker, putWorkflow } = createWorkerWithSpy();
    const wf = new WorkflowDeclaration({ name: 'dn-none' });
    wf.task({ name: 'step', fn: async () => undefined });

    await worker.registerWorkflow(wf);

    const [[req]] = putWorkflow.mock.calls;
    expect(req.displayName).toBeUndefined();
    expect(req.tasks[0].displayName).toBeUndefined();
  });

  it('threads both workflow- and task-level displayName for a standalone task', async () => {
    const { worker, putWorkflow } = createWorkerWithSpy();
    const task = new TaskWorkflowDeclaration({
      name: 'dn-standalone',
      displayName: 'input.run',
      fn: async () => undefined,
    });

    await worker.registerWorkflow(task);

    const [[req]] = putWorkflow.mock.calls;
    // On a single-task workflow the one CEL expression names the run via both the
    // workflow- and task-level fields; the engine's step→workflow precedence resolves
    // to the same value.
    expect(req.displayName).toBe('input.run');
    expect(req.tasks[0].displayName).toBe('input.run');
  });
});
