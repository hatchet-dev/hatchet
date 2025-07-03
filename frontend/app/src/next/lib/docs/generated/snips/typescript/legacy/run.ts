import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  const res = await hatchet.run<{ Message: string }, { step2: string }>(simple, {\n    Message: 'hello',\n  });\n\n  console.log(res.step2);\n}\n\nif (require.main === module) {\n  main();\n}\n",
  source: 'out/typescript/legacy/run.ts',
  blocks: {},
  highlights: {},
};

export default snippet;
