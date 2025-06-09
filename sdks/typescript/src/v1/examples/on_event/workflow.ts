import { hatchet } from '../hatchet-client';

export type Input = {
  Message: string;
  ShouldSkip: boolean;
};

export const SIMPLE_EVENT = 'simple-event:create';

type LowerOutput = {
  lower: {
    TransformedMessage: string;
  };
};

// > Run workflow on event
export const lower = hatchet.workflow<Input, LowerOutput>({
  name: 'lower',
  // 👀 Declare the event that will trigger the workflow
  onEvents: ['simple-event:create'],
});
// !!

// > Workflow with filter
export const lowerWithFilter = hatchet.workflow<Input, LowerOutput>({
  name: 'lower',
  // 👀 Declare the event that will trigger the workflow
  onEvents: ['simple-event:create'],
  defaultFilters: [
    {
      expression: "true",
      scope: "example-scope",
      payload: {
          mainCharacter: "Anna",
          supportingCharacter: "Stiva",
          location: "Moscow",
      }
    }
  ]
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

// > Accessing the filter payload
lowerWithFilter.task({
  name: "lowerWithFilter",
  fn: (input, ctx) => {
    console.log(ctx.filterPayload())
  }
})
// !!
