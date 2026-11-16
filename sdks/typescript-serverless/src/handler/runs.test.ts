import { describe, expect, it } from 'vitest';
import { WorkflowRunEvent, WorkflowRunEventType } from '../generated/proto/dispatcher';
import { outputsOf } from './runs';

function finished(results: Array<{ taskName: string; output?: string }>): WorkflowRunEvent {
  return WorkflowRunEvent.create({
    workflowRunId: 'run-1',
    eventType: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FINISHED,
    results: results.map((result) => ({
      taskRunExternalId: crypto.randomUUID(),
      jobRunId: crypto.randomUUID(),
      ...result,
    })),
  });
}

describe('outputsOf', () => {
  it('keys each task output by name, with {} for an empty output', () => {
    const outputs = outputsOf<Record<string, unknown>>(
      finished([{ taskName: 'a', output: '{"x":1}' }, { taskName: 'b' }])
    );

    expect(outputs).toEqual({ a: { x: 1 }, b: {} });
    expect(Object.getPrototypeOf(outputs)).toBe(Object.prototype);
    expect(outputsOf(finished([{ taskName: 'a', output: '{"x":1}' }]), 'a')).toEqual({ x: 1 });
  });

  it('refuses a task name that would read as a prototype member', () => {
    for (const taskName of ['__proto__', 'constructor', 'prototype']) {
      expect(() => outputsOf(finished([{ taskName, output: '{"polluted":true}' }]))).toThrow(
        /reserved/
      );
    }

    expect(({} as { polluted?: boolean }).polluted).toBeUndefined();
  });
});
