import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "import { simple } from './flow-control';\n\n// > Schedules and Crons\nconst tomorrow = new Date(Date.now() + 1000 * 60 * 60 * 24);\nconst scheduled = simple.schedule(tomorrow, {\n  Message: 'Hello, World!',\n});\n\nconst cron = simple.cron('every-day', '0 0 * * *', {\n  Message: 'Hello, World!',\n});\n",
  "source": "out/typescript/landing_page/scheduling.ts",
  "blocks": {
    "schedules_and_crons": {
      "start": 4,
      "stop": 11
    }
  },
  "highlights": {}
};

export default snippet;
