// Generated from /Users/gabrielruttner/dev/hatchet/examples/typescript/child_workflows/run.ts
export const content = "import { parent } from './workflow';\n\nasync function main() {\n  const res = await parent.run({\n    N: 10,\n  });\n\n  // eslint-disable-next-line no-console\n  console.log(res.Result);\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      // eslint-disable-next-line no-console\n      console.error('Error:', error);\n      process.exit(1);\n    });\n}\n";
export const language = "ts";
