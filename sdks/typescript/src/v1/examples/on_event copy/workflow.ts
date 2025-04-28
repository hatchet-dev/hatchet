import { hatchet } from '../hatchet-client';

export type Input = {
  Message: string;
};

type LowerOutput = {
  lower: {
    TransformedMessage: string;
  };
};

// > Run workflow on event
export const lower = hatchet.workflow<Input, LowerOutput>({
  name: 'lower',
  on: {
    // 👀 Declare the event that will trigger the workflow
    event: 'simple-event:create',
  },
});
// !!

lower.task({
  name: 'lower',
  fn: (input) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});

type UpperOutput = {
  upper: {
    TransformedMessage: string;
  };
};

export const upper = hatchet.workflow<Input, UpperOutput>({
  name: 'upper',
  on: {
    event: 'simple-event:create',
  },
});

upper.task({
  name: 'upper',
  fn: (input) => {
    return {
      TransformedMessage: input.Message.toUpperCase(),
    };
  },
});
