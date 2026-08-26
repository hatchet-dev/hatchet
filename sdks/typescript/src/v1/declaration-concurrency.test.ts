import { WorkflowDeclaration } from './declaration';
import { ConcurrencyLimitStrategy } from './task';

// Simple smoke tests confirming a task can be registered with the CANCEL_EXCEPT_NEWEST/
// CANCEL_EXCEPT_OLDEST concurrency strategies - not a scenario test of their runtime behavior.
describe('task concurrency registration', () => {
  it('registers a task with CANCEL_EXCEPT_NEWEST', () => {
    const wf = new WorkflowDeclaration({ name: 'concurrency-except-newest-test' });

    const task = wf.task({
      name: 'except-newest-task',
      fn: async () => undefined,
      concurrency: [
        {
          expression: 'input.group',
          maxRuns: 1,
          limitStrategy: ConcurrencyLimitStrategy.CANCEL_EXCEPT_NEWEST,
        },
      ],
    });

    expect(task.concurrency).toEqual([
      {
        expression: 'input.group',
        maxRuns: 1,
        limitStrategy: ConcurrencyLimitStrategy.CANCEL_EXCEPT_NEWEST,
      },
    ]);
  });

  it('registers a task with CANCEL_EXCEPT_OLDEST', () => {
    const wf = new WorkflowDeclaration({ name: 'concurrency-except-oldest-test' });

    const task = wf.task({
      name: 'except-oldest-task',
      fn: async () => undefined,
      concurrency: [
        {
          expression: 'input.group',
          maxRuns: 1,
          limitStrategy: ConcurrencyLimitStrategy.CANCEL_EXCEPT_OLDEST,
        },
      ],
    });

    expect(task.concurrency).toEqual([
      {
        expression: 'input.group',
        maxRuns: 1,
        limitStrategy: ConcurrencyLimitStrategy.CANCEL_EXCEPT_OLDEST,
      },
    ]);
  });
});
