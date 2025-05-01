import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { firstTask } from './workflows/first-task';\n\nasync function main() {\n  const worker = await hatchet.worker('first-worker', {\n    workflows: [firstTask],\n    slots: 10,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/quickstart/worker.ts',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
