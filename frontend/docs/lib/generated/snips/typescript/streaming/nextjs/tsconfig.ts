import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "unknown",
  "content": "{\n  \"extends\": \"../../../../../tsconfig.json\",\n  \"compilerOptions\": {\n    \"target\": \"ES2017\",\n    \"lib\": [\"dom\", \"dom.iterable\", \"esnext\"],\n    \"allowJs\": true,\n    \"skipLibCheck\": true,\n    \"strict\": false,\n    \"noEmit\": true,\n    \"incremental\": true,\n    \"module\": \"esnext\",\n    \"esModuleInterop\": true,\n    \"moduleResolution\": \"node\",\n    \"resolveJsonModule\": true,\n    \"isolatedModules\": true,\n    \"jsx\": \"preserve\",\n    \"plugins\": [\n      {\n        \"name\": \"next\"\n      }\n    ]\n  },\n  \"include\": [\"next-env.d.ts\", \".next/types/**/*.ts\", \"**/*.ts\", \"**/*.tsx\"],\n  \"exclude\": [\"node_modules\"]\n}\n",
  "source": "out/typescript/streaming/nextjs/tsconfig.json",
  "blocks": {},
  "highlights": {}
};

export default snippet;
