// Usage in a DIFFERENT file from the wrapper (see ./builder.ts). The DAG renders
// on `ordersDag` here: the workspace annotation cache resolves `createWorkflow`
// as an `@hatchet-workflow` factory, so this usage is treated as a workflow.

import { createWorkflow } from './builder';

const ordersDag = createWorkflow({ name: 'orders-dag' });

const start = ordersDag.task({ name: 'start' });
const branchA = ordersDag.task({ name: 'branch-a', parents: [start] });
const branchB = ordersDag.task({ name: 'branch-b', parents: [start] });
const join = ordersDag.task({ name: 'join', parents: [branchA, branchB] });
