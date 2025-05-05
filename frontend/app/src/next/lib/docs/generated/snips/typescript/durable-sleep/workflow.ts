import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "// import sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\nexport const durableSleep = hatchet.workflow({\n  name: 'durable-sleep',\n});\n\n// > Durable Sleep\ndurableSleep.durableTask({\n  name: 'durable-sleep',\n  executionTimeout: '10m',\n  fn: async (_, ctx) => {\n    console.log('sleeping for 5s');\n    const sleepRes = await ctx.sleepFor('5s');\n    console.log('done sleeping for 5s', sleepRes);\n\n    return {\n      Value: 'done',\n    };\n  },\n});\n",
  source: 'out/typescript/durable-sleep/workflow.ts',
  blocks: {
    durable_sleep: {
      start: 9,
      stop: 21,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
