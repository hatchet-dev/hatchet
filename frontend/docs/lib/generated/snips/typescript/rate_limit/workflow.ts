import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { RateLimitDuration } from '@hatchet-dev/typescript-sdk/protoc/v1/workflows';\nimport { hatchet } from '../hatchet-client';\n\n// > Upsert Rate Limit\nhatchet.ratelimits.upsert({\n  key: 'api-service-rate-limit',\n  limit: 10,\n  duration: RateLimitDuration.SECOND,\n});\n\n// > Static\nconst RATE_LIMIT_KEY = 'api-service-rate-limit';\n\nconst task1 = hatchet.task({\n  name: 'task1',\n  rateLimits: [\n    {\n      staticKey: RATE_LIMIT_KEY,\n      units: 1,\n    },\n  ],\n  fn: (input) => {\n    console.log('executed task1');\n  },\n});\n\n\n// > Dynamic\nconst task2 = hatchet.task({\n  name: 'task2',\n  fn: (input: { userId: string }) => {\n    console.log('executed task2 for user: ', input.userId);\n  },\n  rateLimits: [\n    {\n      dynamicKey: 'input.userId',\n      units: 1,\n      limit: 10,\n      duration: RateLimitDuration.MINUTE,\n    },\n  ],\n});\n",
  "source": "out/typescript/rate_limit/workflow.ts",
  "blocks": {
    "upsert_rate_limit": {
      "start": 5,
      "stop": 9
    },
    "static": {
      "start": 12,
      "stop": 26
    },
    "dynamic": {
      "start": 29,
      "stop": 42
    }
  },
  "highlights": {}
};

export default snippet;
