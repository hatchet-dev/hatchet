import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { durableEvent } from './workflow';\n\nasync function main() {\n  const timeStart = Date.now();\n  const res = await durableEvent.run({});\n  const timeEnd = Date.now();\n  console.log(`Time taken: ${timeEnd - timeStart}ms`);\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n",
  "source": "out/typescript/durable-event/run.ts",
  "blocks": {},
  "highlights": {}
};

export default snippet;
