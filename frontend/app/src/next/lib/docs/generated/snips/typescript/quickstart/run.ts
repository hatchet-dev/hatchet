import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { firstTask } from './workflows/first-task';\n\nasync function main() {\n  const res = await firstTask.run({\n    Message: 'Hello World!',\n  });\n\n  console.log(\n    'Finished running task, and got the transformed message! The transformed message is:',\n    res.TransformedMessage\n  );\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
  source: 'out/typescript/quickstart/run.ts',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
