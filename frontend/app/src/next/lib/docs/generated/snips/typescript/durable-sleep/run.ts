import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { durableSleep } from \'./workflow\';\n\nasync function main() {\n  const timeStart = Date.now();\n  const res = await durableSleep.run({});\n  const timeEnd = Date.now();\n  console.log(`Time taken: ${timeEnd - timeStart}ms`);\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      console.error(\'Error:\', error);\n      process.exit(1);\n    });\n}\n',
  'source': 'out/typescript/durable-sleep/run.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
