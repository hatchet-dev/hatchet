import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { hatchet } from '../hatchet-client';\nimport { streaming_task } from './workflow';\n\n\nasync function main() {\n  const worker = await hatchet.worker('streaming-worker', {\n    workflows: [streaming_task],\n  });\n\n  await worker.start();\n}\n\nif (require.main === module) {\n  main();\n}\n",
  "source": "out/typescript/streaming/worker.ts",
  "blocks": {},
  "highlights": {}
};

export default snippet;
