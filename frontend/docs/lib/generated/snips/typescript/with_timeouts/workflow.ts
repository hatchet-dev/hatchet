import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "// > Declaring a Task\nimport sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\n\n// > Execution Timeout\nexport const withTimeouts = hatchet.task({\n  name: 'with-timeouts',\n  // time the task can wait in the queue before it is cancelled\n  scheduleTimeout: '10s',\n  // time the task can run before it is cancelled\n  executionTimeout: '10s',\n  fn: async (input: SimpleInput, ctx) => {\n    // wait 15 seconds\n    await sleep(15000);\n\n    // get the abort controller\n    const { abortController } = ctx;\n\n    // if the abort controller is aborted, throw an error\n    if (abortController.signal.aborted) {\n      throw new Error('cancelled');\n    }\n\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n\n// > Refresh Timeout\nexport const refreshTimeout = hatchet.task({\n  name: 'refresh-timeout',\n  executionTimeout: '10s',\n  scheduleTimeout: '10s',\n  fn: async (input: SimpleInput, ctx) => {\n    // adds 15 seconds to the execution timeout\n    ctx.refreshTimeout('15s');\n    await sleep(15000);\n\n    // get the abort controller\n    const { abortController } = ctx;\n\n    // now this condition will not be met\n    // if the abort controller is aborted, throw an error\n    if (abortController.signal.aborted) {\n      throw new Error('cancelled');\n    }\n\n    return {\n      TransformedMessage: input.Message.toLowerCase(),\n    };\n  },\n});\n",
  "source": "out/typescript/with_timeouts/workflow.ts",
  "blocks": {
    "execution_timeout": {
      "start": 11,
      "stop": 33
    },
    "refresh_timeout": {
      "start": 36,
      "stop": 58
    }
  },
  "highlights": {}
};

export default snippet;
