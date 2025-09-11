import { hatchet } from '../hatchet-client';
import { lower, SIMPLE_EVENT } from './workflow';

// > Create a filter
hatchet.filters.create({
  workflowId: lower.id,
  expression: 'input.ShouldSkip == false',
  scope: 'foobarbaz',
  payload: {
    main_character: 'Anna',
    supporting_character: 'Stiva',
    location: 'Moscow',
  },
});
// !!

// > Skip a run
hatchet.events.push(
  SIMPLE_EVENT,
  {
    Message: 'hello',
    ShouldSkip: true,
  },
  {
    scope: 'foobarbaz',
  }
);
// !!

// > Trigger a run
hatchet.events.push(
  SIMPLE_EVENT,
  {
    Message: 'hello',
    ShouldSkip: false,
  },
  {
    scope: 'foobarbaz',
  }
);
// !!
