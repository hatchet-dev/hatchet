import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { multiConcurrency } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-concurrency-worker', {\n    workflows: [multiConcurrency],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/multiple_wf_concurrency/worker.ts',
  blocks: {},
  highlights: {},
};

export default snippet;
