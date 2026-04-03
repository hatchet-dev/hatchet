import WorkflowRunRef from '@hatchet/util/workflow-run-ref';
import { BaseWorkflowDeclaration, RunOpts } from './declaration';
import { IHatchetClient } from './client/client.interface';

type TestInput = { value: number };
type TestOutput = { ok: boolean; value: number };
type RunRequest = { input: TestInput; options?: RunOpts };

function createTestClient(runWorkflows: jest.Mock): IHatchetClient {
  return {
    admin: {
      runWorkflows,
      runWorkflow: jest.fn(),
      logger: { warn: jest.fn() },
    },
  } as unknown as IHatchetClient;
}

function createResolvedRunRef(run: RunRequest, index: number): WorkflowRunRef<TestOutput> {
  return {
    id: `run-${index + 1}`,
    result: jest.fn().mockResolvedValue({
      ok: true,
      value: run.input.value,
    }),
  } as unknown as WorkflowRunRef<TestOutput>;
}

function createPendingRunRef(index: number): WorkflowRunRef<object> {
  return {
    id: `run-${index + 1}`,
  } as unknown as WorkflowRunRef<object>;
}

describe('BaseWorkflowDeclaration run', () => {
  function createWorkflow() {
    const runWorkflows = jest
      .fn()
      .mockImplementation(async (runs: RunRequest[]) => runs.map(createResolvedRunRef));

    const workflow = new BaseWorkflowDeclaration<TestInput, TestOutput>(
      { name: 'test-workflow' },
      createTestClient(runWorkflows)
    );

    return { workflow, runWorkflows };
  }

  it('keeps shared options behavior for input arrays', async () => {
    const { workflow, runWorkflows } = createWorkflow();

    const sharedOptions: RunOpts = {
      additionalMetadata: { source: 'shared' },
    };

    const result = await workflow.run([{ value: 1 }, { value: 2 }], sharedOptions);

    expect(result).toEqual([
      { ok: true, value: 1 },
      { ok: true, value: 2 },
    ]);

    expect(runWorkflows).toHaveBeenCalledTimes(1);
    const [[workflowCalls]] = runWorkflows.mock.calls;
    const [firstCall, secondCall] = workflowCalls;

    expect(firstCall.options?.additionalMetadata).toEqual({ source: 'shared' });
    expect(secondCall.options?.additionalMetadata).toEqual({ source: 'shared' });
  });

  it('supports per-item options arrays for input arrays', async () => {
    const { workflow, runWorkflows } = createWorkflow();

    const result = await workflow.run(
      [{ value: 1 }, { value: 2 }],
      [{ additionalMetadata: { source: 'first' } }, { additionalMetadata: { source: 'second' } }]
    );

    expect(result).toEqual([
      { ok: true, value: 1 },
      { ok: true, value: 2 },
    ]);

    expect(runWorkflows).toHaveBeenCalledTimes(1);
    const [[workflowCalls]] = runWorkflows.mock.calls;
    const [firstCall, secondCall] = workflowCalls;

    expect(firstCall.options?.additionalMetadata).toEqual({ source: 'first' });
    expect(secondCall.options?.additionalMetadata).toEqual({ source: 'second' });
  });

  it('throws when options array length does not match input length', async () => {
    const { workflow } = createWorkflow();

    await expect(
      workflow.run([{ value: 1 }, { value: 2 }], [{ additionalMetadata: { source: 'first' } }])
    ).rejects.toThrow('options array length must match input array length');
  });
});

describe('BaseWorkflowDeclaration runNoWait', () => {
  function createWorkflow() {
    const runWorkflows = jest
      .fn()
      .mockImplementation(async (runs: RunRequest[]) => runs.map((_, i) => createPendingRunRef(i)));

    const workflow = new BaseWorkflowDeclaration<TestInput, { ok: boolean }>(
      { name: 'test-workflow' },
      createTestClient(runWorkflows)
    );

    return { workflow, runWorkflows };
  }

  it('keeps shared options behavior for input arrays', async () => {
    const { workflow, runWorkflows } = createWorkflow();

    const sharedOptions: RunOpts = {
      additionalMetadata: { source: 'shared' },
    };

    await workflow.runNoWait([{ value: 1 }, { value: 2 }], sharedOptions);

    expect(runWorkflows).toHaveBeenCalledTimes(1);
    const [[workflowCalls]] = runWorkflows.mock.calls;
    const [firstCall, secondCall] = workflowCalls;

    expect(firstCall.options?.additionalMetadata).toEqual({ source: 'shared' });
    expect(secondCall.options?.additionalMetadata).toEqual({ source: 'shared' });
  });

  it('supports per-item options arrays for input arrays', async () => {
    const { workflow, runWorkflows } = createWorkflow();

    await workflow.runNoWait(
      [{ value: 1 }, { value: 2 }],
      [{ additionalMetadata: { source: 'first' } }, { additionalMetadata: { source: 'second' } }]
    );

    expect(runWorkflows).toHaveBeenCalledTimes(1);
    const [[workflowCalls]] = runWorkflows.mock.calls;
    const [firstCall, secondCall] = workflowCalls;

    expect(firstCall.options?.additionalMetadata).toEqual({ source: 'first' });
    expect(secondCall.options?.additionalMetadata).toEqual({ source: 'second' });
  });

  it('throws when options array length does not match input length', async () => {
    const { workflow } = createWorkflow();

    await expect(
      workflow.runNoWait(
        [{ value: 1 }, { value: 2 }],
        [{ additionalMetadata: { source: 'first' } }]
      )
    ).rejects.toThrow('options array length must match input array length');
  });
});
