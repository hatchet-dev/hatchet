import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport axios from 'axios';\nimport { hatchet } from '../hatchet-client';\n\n// > Declaring a Task\nexport const cancellation = hatchet.task({\n  name: 'cancellation',\n  fn: async (_, { cancelled }) => {\n    await sleep(10 * 1000);\n\n    if (cancelled) {\n      throw new Error('Task was cancelled');\n    }\n\n    return {\n      Completed: true,\n    };\n  },\n});\n\n// > Abort Signal\nexport const abortSignal = hatchet.task({\n  name: 'abort-signal',\n  fn: async (_, { abortController }) => {\n    try {\n      const response = await axios.get('https://api.example.com/data', {\n        signal: abortController.signal,\n      });\n      // Handle the response\n    } catch (error) {\n      if (axios.isCancel(error)) {\n        // Request was canceled\n        console.log('Request canceled');\n      } else {\n        // Handle other errors\n      }\n    }\n  },\n});\n\n// see ./worker.ts and ./run.ts for how to run the workflow\n",
  source: 'out/typescript/cancellations/workflow.ts',
  blocks: {
    declaring_a_task: {
      start: 6,
      stop: 19,
    },
    abort_signal: {
      start: 22,
      stop: 39,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
