import { RateLimitDuration } from '@hatchet/protoc/v1/workflows';
import { hatchet } from '../hatchet-client';

// > Upsert Rate Limit
hatchet.ratelimits.upsert({
  key: 'api-service-rate-limit',
  limit: 10,
  duration: RateLimitDuration.SECOND,
});
// !!

// > Static
const RATE_LIMIT_KEY = 'api-service-rate-limit';

const task1 = hatchet.task({
  name: 'task1',
  rateLimits: [
    {
      staticKey: RATE_LIMIT_KEY,
      units: 1,
    },
  ],
  fn: (input) => {
    console.log('executed task1');
  },
});

// !!

// > Dynamic
const task2 = hatchet.task({
  name: 'task2',
  fn: (input: { userId: string }) => {
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
