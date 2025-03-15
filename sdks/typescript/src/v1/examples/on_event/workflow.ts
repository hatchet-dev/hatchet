import { hatchet } from '../client';

export const SIMPLE_EVENT = 'simple-event:create';

type Input = {
  Message: string;
};

type LowerOutput = {
  lower: {
    TransformedMessage: string;
  };
};

export const lower = hatchet.workflow<Input, LowerOutput>({
  name: 'lower',
  on: {
    event: SIMPLE_EVENT,
  },
});

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
    event: SIMPLE_EVENT,
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
