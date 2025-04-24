// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/deep/run.ts
export const content = "import { parent } from './workflow';\n\nasync function main() {\n  const res = await parent.run({\n    Message: 'hello',\n    N: 5,\n  });\n\n  // eslint-disable-next-line no-console\n  console.log(res.parent.Sum);\n}\n\nif (require.main === module) {\n  main()\n    // eslint-disable-next-line no-console\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n";
export const language = "ts";
