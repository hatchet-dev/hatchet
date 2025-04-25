import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

// ❓ Route tasks to workers with matching labels
export const simple = hatchet.task({
  name: 'simple',
  desiredWorkerLabels: {
    cpu: {
      value: '2x',
    },
  },
  fn: (input: SimpleInput) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

hatchet.worker('task-routing-worker', {
  workflows: [simple],
  labels: {
    cpu: process.env.CPU_LABEL,
  },
});
