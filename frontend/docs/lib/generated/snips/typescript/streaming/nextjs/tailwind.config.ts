import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "unknown",
  "content": "export default {\n  content: [\n    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',\n    './src/components/**/*.{js,ts,jsx,tsx,mdx}',\n    './src/app/**/*.{js,ts,jsx,tsx,mdx}',\n  ],\n  theme: {\n    extend: {\n      colors: {\n        background: 'var(--background)',\n        foreground: 'var(--foreground)',\n      },\n    },\n  },\n  plugins: [],\n};\n",
  "source": "out/typescript/streaming/nextjs/tailwind.config.js",
  "blocks": {},
  "highlights": {}
};

export default snippet;
