import {
  RunListenerClient,
  StepRunEvent,
} from '@hatchet/clients/listeners/run-listener/child-listener-client';
import { WorkflowRunEventType } from '../protoc/dispatcher';
import WorkflowRunRef from './workflow-run-ref';

describe('WorkflowRunRef', () => {
  it('rejects failed runs with an Error containing only failed step messages', async () => {
    const finishedEvent = {
      eventType: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FINISHED,
      results: [
        { taskName: 'successful-step', error: undefined, output: '{}' },
        { taskName: 'failed-step', error: 'step failed', output: '{}' },
      ],
    } as unknown as StepRunEvent;
    const client = {
      get: jest.fn().mockResolvedValue({
        stream: async function* () {
          yield finishedEvent;
        },
      }),
    } as unknown as RunListenerClient;
    const workflowRun = new WorkflowRunRef('workflow-run-id', client);

    expect.assertions(2);

    try {
      await workflowRun.result();
    } catch (error) {
      expect(error).toBeInstanceOf(Error);
      expect((error as Error).message).toBe('step failed');
    }
  });
});
