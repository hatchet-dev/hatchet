import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\n\n// (optional) Define the input type for the workflow\nexport type SimpleInput = {\n  Message: string;\n};\nasync function main() {\n  // > Declaring a Task\n  const simple = hatchet.task({\n    name: 'simple',\n    fn: (input: SimpleInput) => {\n      return {\n        TransformedMessage: input.Message.toLowerCase(),\n      };\n    },\n  });\n\n  // > Running a Task\n  const result = await simple.run({ Message: 'Hello, World!' });\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/landing_page/queues.ts',
  blocks: {
    declaring_a_task: {
      start: 9,
      stop: 16,
    },
    running_a_task: {
      start: 19,
      stop: 19,
    },
  },
  highlights: {},
};

export default snippet;
