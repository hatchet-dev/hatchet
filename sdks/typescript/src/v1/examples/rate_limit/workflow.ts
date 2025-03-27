import { RateLimitDuration } from '@hatchet/protoc/v1/workflows';
import { hatchet } from '../hatchet-client';

// ❓ Workflow
type RateLimitInput = {
  userId: string;
};

export const rateLimitWorkflow = hatchet.workflow<RateLimitInput>({
  name: 'RateLimitWorkflow',
});

// !!

// ❓ Static
const RATE_LIMIT_KEY = 'test-limit';

const task1 = rateLimitWorkflow.task({
  name: 'task1',
  fn: (input) => {
    console.log('executed task1');
  },
  rateLimits: [
    {
      staticKey: RATE_LIMIT_KEY,
      units: 1,
    },
  ],
});

// !!

// ❓ Dynamic
const task2 = rateLimitWorkflow.task({
  name: 'task2',
  fn: (input) => {
    console.log('executed task2 for user: ', input.userId);
  },
  rateLimits: [
    {
      dynamicKey: 'input.userId',
      units: 1,
      limit: 10,
      duration: RateLimitDuration.MINUTE,
    },
  ],
});

// !!
