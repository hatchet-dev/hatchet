import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "// > Declaring a Worker\nimport { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\nimport { parent, child } from './workflow-with-child';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-worker', {\n    // 👀 Declare the workflows that the worker can execute\n    workflows: [simple, parent, child],\n    // 👀 Declare the number of concurrent task runs the worker can accept\n    slots: 100,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/simple/worker.ts',
  blocks: {
    declaring_a_worker: {
      start: 2,
      stop: 19,
    },
  },
  highlights: {},
};

export default snippet;
