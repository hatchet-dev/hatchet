import { roleAllows } from './role-permissions';
import assert from 'node:assert/strict';
import { test } from 'node:test';

const roles = [
  {
    name: 'ADMIN',
    label: 'Admin',
    description: '',
    operations: ['ApiTokenCreate', 'WorkflowRunList'],
  },
  {
    name: 'VIEWER',
    label: 'Viewer',
    description: '',
    operations: ['WorkflowRunList'],
  },
];

test('a role allows exactly the operations the server lists for it', () => {
  assert.equal(roleAllows(roles, 'ADMIN', 'ApiTokenCreate'), true);
  assert.equal(roleAllows(roles, 'VIEWER', 'ApiTokenCreate'), false);
  assert.equal(roleAllows(roles, undefined, 'WorkflowRunList'), false);
  assert.equal(roleAllows([], 'ADMIN', 'ApiTokenCreate'), false);
});
