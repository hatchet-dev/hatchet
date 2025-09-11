import { hatchet } from '../hatchet-client';

// (optional) Define the input type for the workflow
export type SimpleInput = {
  Message: string;
};

// > Trigger on an event
export const simple = hatchet.task({
  name: 'simple',
  onEvents: ['user:created'],
  fn: (input: SimpleInput) => {
    // ...
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
// !!
