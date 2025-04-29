import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\nimport { failureWorkflow } from \'./workflow\';\n\nasync function main() {\n  const worker = await hatchet.worker(\'always-fail-worker\', {\n    workflows: [failureWorkflow],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/on_failure/worker.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
