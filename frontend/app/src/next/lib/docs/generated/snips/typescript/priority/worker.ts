import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { priorityTasks } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('priority-worker', {\n    workflows: [...priorityTasks],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/priority/worker.ts',
  blocks: {},
  highlights: {},
};

export default snippet;
