import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { multiConcurrency } from './workflow';\n\nasync function main() {\n  const res = await multiConcurrency.run([\n    {\n      Message: 'Hello World',\n      GroupKey: 'A',\n    },\n    {\n      Message: 'Goodbye Moon',\n      GroupKey: 'A',\n    },\n    {\n      Message: 'Hello World B',\n      GroupKey: 'B',\n    },\n  ]);\n  console.log(res[0]['to-lower'].TransformedMessage);\n  console.log(res[1]['to-lower'].TransformedMessage);\n  console.log(res[2]['to-lower'].TransformedMessage);\n}\n\nif (require.main === module) {\n  main().then(() => process.exit(0));\n}\n",
  source: 'out/typescript/multiple_wf_concurrency/run.ts',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
