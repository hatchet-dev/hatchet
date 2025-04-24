// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/dag_match_condition/run.ts
export const content = "/* eslint-disable no-console */\nimport { dagWithConditions } from './workflow';\n\nasync function main() {\n  const res = await dagWithConditions.run({});\n\n  console.log(res['first-task'].Completed);\n  console.log(res['second-task'].Completed);\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n";
export const language = "ts";
