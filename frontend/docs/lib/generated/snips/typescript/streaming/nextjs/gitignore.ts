import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "unknown",
  "content": "# See https://help.github.com/articles/ignoring-files/ for more about ignoring files.\n\n# dependencies\n/node_modules\n/.pnp\n.pnp.*\n.yarn/*\n!.yarn/patches\n!.yarn/plugins\n!.yarn/releases\n!.yarn/versions\n\n# testing\n/coverage\n\n# next.js\n/.next/\n/out/\n\n# production\n/build\n\n# misc\n.DS_Store\n*.pem\n\n# debug\nnpm-debug.log*\nyarn-debug.log*\nyarn-error.log*\n.pnpm-debug.log*\n\n# env files (can opt-in for committing if needed)\n.env*\n\n# vercel\n.vercel\n\n# typescript\n*.tsbuildinfo\nnext-env.d.ts\n",
  "source": "out/typescript/streaming/nextjs/.gitignore",
  "blocks": {},
  "highlights": {}
};

export default snippet;
