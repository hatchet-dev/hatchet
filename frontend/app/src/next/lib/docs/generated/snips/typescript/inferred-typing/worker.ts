import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { declaredType, inferredType, inferredTypeDurable, crazyWorkflow } from './workflow';\n\nasync function main() {\n  const worker = await hatchet.worker('simple-worker', {\n    workflows: [declaredType, inferredType, inferredTypeDurable, crazyWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/inferred-typing/worker.ts',
  blocks: {},
  highlights: {},
};

export default snippet;
