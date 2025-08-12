import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { durableEvent } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('durable-event-worker', {\n    workflows: [durableEvent],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/durable-event/worker.ts',
  blocks: {},
  highlights: {},
};

export default snippet;
