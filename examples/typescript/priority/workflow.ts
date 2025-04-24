/* eslint-disable no-console */
import { Priority } from '@hatchet-dev/typescript-sdk';
import { hatchet } from '../hatchet-client';

// ❓ Simple Task Priority
export const priority = hatchet.task({
  name: 'priority',
  defaultPriority: Priority.MEDIUM,
  fn: async (_, ctx) => {
    return {
      priority: ctx.priority(),
    };
  },
});
// !!

// ❓ Task Priority in a Workflow
export const priorityWf = hatchet.workflow({
  name: 'priorityWf',
  defaultPriority: Priority.LOW,
});
// !!

priorityWf.task({
  name: 'child-medium',
  fn: async (_, ctx) => {
    return {
      priority: ctx.priority(),
    };
  },
});

priorityWf.task({
  name: 'child-high',
  // will inherit the default priority from the workflow
  fn: async (_, ctx) => {
    return {
      priority: ctx.priority(),
    };
  },
});

export const priorityTasks = [priority, priorityWf];
