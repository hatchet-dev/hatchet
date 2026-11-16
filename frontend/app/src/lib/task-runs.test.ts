import { isStandaloneRun } from './task-runs';
import assert from 'node:assert/strict';
import { test } from 'node:test';

const run = (id: string, workflowRunExternalId: string) => ({
  metadata: { id, createdAt: '', updatedAt: '' },
  workflowRunExternalId,
});

test('a row whose run id is its own id is a standalone run', () => {
  assert.equal(isStandaloneRun(run('run-a', 'run-a')), true);
});

test('a task inside a DAG run is not a standalone run', () => {
  assert.equal(isStandaloneRun(run('task-a', 'dag-run-a')), false);
});
