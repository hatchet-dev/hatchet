import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "unknown",
  "content": "\nimport path from 'path';\nimport { fileURLToPath } from 'url';\n\nconst __filename = fileURLToPath(import.meta.url);\nconst __dirname = path.dirname(__filename);\n\nconst nextConfig = {\n  webpack: (config) => {\n    config.resolve.alias = {\n      ...config.resolve.alias,\n      '@hatchet/v1': path.resolve(__dirname, '../../../../../../src/v1'),\n      '@hatchet-dev/typescript-sdk': path.resolve(__dirname, '../../../../../../src'),\n    };\n    return config;\n  },\n};\n\nexport default nextConfig;\n",
  "source": "out/typescript/streaming/nextjs/next.config.js",
  "blocks": {},
  "highlights": {}
};

export default snippet;
