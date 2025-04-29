import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "\nimport { Or } from '@hatchet-dev/typescript-sdk/v1/conditions';\nimport { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\nasync function main() {\n  // > Declaring a Durable Task\n  const simple = hatchet.durableTask({\n    name: 'simple',\n    fn: async (input: SimpleInput, ctx) => {\n      await ctx.waitFor(\n        Or(\n          {\n            eventKey: 'user:pay',\n            expression: 'input.Status == 'PAID'',\n          },\n          {\n            sleepFor: '24h',\n          }\n        )\n      );\n\n      return {\n        TransformedMessage: input.Message.toLowerCase(),\n      };\n    },\n  });\n  \n\n  // > Running a Task\n  const result = await simple.run({ Message: 'Hello, World!' });\n  \n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/landing_page/durable-excution.ts',
  blocks: {
    declaring_a_durable_task: {
      start: 10,
      stop: 29,
    },
    running_a_task: {
      start: 32,
      stop: 32,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
