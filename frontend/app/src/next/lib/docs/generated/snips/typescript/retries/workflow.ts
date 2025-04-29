import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "\nimport { hatchet } from '../hatchet-client';\n\n// > Simple Step Retries\nexport const retries = hatchet.task({\n  name: 'retries',\n  retries: 3,\n  fn: async (_, ctx) => {\n    throw new Error('intentional failure');\n  },\n});\n\n\n// > Retries with Count\nexport const retriesWithCount = hatchet.task({\n  name: 'retriesWithCount',\n  retries: 3,\n  fn: async (_, ctx) => {\n    // > Get the current retry count\n    const retryCount = ctx.retryCount();\n\n    console.log(`Retry count: ${retryCount}`);\n\n    if (retryCount < 2) {\n      throw new Error('intentional failure');\n    }\n\n    return {\n      message: 'success',\n    };\n  },\n});\n\n\n// > Retries with Backoff\nexport const withBackoff = hatchet.task({\n  name: 'withBackoff',\n  retries: 10,\n  backoff: {\n    // 👀 Maximum number of seconds to wait between retries\n    maxSeconds: 10,\n    // 👀 Factor to increase the wait time between retries.\n    // This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit\n    factor: 2,\n  },\n  fn: async () => {\n    throw new Error('intentional failure');\n  },\n});\n\n",
  source: 'out/typescript/retries/workflow.ts',
  blocks: {
    simple_step_retries: {
      start: 4,
      stop: 10,
    },
    get_the_current_retry_count: {
      start: 18,
      stop: 30,
    },
    retries_with_backoff: {
      start: 33,
      stop: 46,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
