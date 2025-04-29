import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { retries } from './workflow';\n\nasync function main() {\n  try {\n    const res = await retries.run({});\n    console.log(res);\n  } catch (e) {\n    console.log('error', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
  source: 'out/typescript/retries/run.ts',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
