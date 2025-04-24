// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/legacy/run.ts
export const content = "import { hatchet } from '../hatchet-client';\nimport { simple } from './workflow';\n\nasync function main() {\n  const res = await hatchet.run<{ Message: string }, { step2: string }>(simple, {\n    Message: 'hello',\n  });\n\n  // eslint-disable-next-line no-console\n  console.log(res.step2);\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
