// ❓ Declaring an External Workflow Reference
import { hatchet } from '../client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

// (optional) Define the output type for the workflow
export type SimpleOutput = {
  'to-lower': {
    TransformedMessage: string;
  };
};

// declare the workflow with the same name as the
// workflow name on the worker
export const simple = hatchet.workflow<SimpleInput, SimpleOutput>({
  name: 'simple',
});

// you can use all the same run methods on the stub
// with full type-safety
simple.run({ Message: 'Hello, World!' });
simple.enqueue({ Message: 'Hello, World!' });
simple.schedule(new Date(), { Message: 'Hello, World!' });
simple.cron('my-cron', '0 0 * * *', { Message: 'Hello, World!' });
// !!
