// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/dag/run.ts
export const content = "import { dag } from './workflow';\n\nasync function main() {\n  const res = await dag.run({\n    Message: 'hello world',\n  });\n\n  // eslint-disable-next-line no-console\n  console.log(res.reverse.Transformed);\n}\n\nif (require.main === module) {\n  main();\n}\n";
export const language = "ts";
