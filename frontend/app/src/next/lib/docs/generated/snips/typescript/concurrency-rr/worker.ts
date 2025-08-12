import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { simpleConcurrency } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-concurrency-worker', {\n    workflows: [simpleConcurrency],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/concurrency-rr/worker.ts',
  blocks: {},
  highlights: {},
};

export default snippet;
