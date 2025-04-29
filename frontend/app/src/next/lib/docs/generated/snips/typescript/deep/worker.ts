import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\nimport { parent, child1, child2, child3, child4, child5 } from \'./workflow\';\n\nasync function main() {\n  const worker = await hatchet.worker(\'simple-worker\', {\n    workflows: [parent, child1, child2, child3, child4, child5],\n    slots: 5000,\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/deep/worker.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
