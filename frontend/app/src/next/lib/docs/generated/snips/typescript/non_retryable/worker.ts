import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\nimport { nonRetryableWorkflow } from \'./workflow\';\n\nasync function main() {\n  const worker = await hatchet.worker(\'no-retry-worker\', {\n    workflows: [nonRetryableWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/non_retryable/worker.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
