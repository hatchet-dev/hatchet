import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    '# > RootLogger\n\n\nimport logging\n\nfrom hatchet_sdk import ClientConfig, Hatchet\n\nlogging.basicConfig(level=logging.INFO)\n\nroot_logger = logging.getLogger()\n\nhatchet = Hatchet(\n    debug=True,\n    config=ClientConfig(\n        logger=root_logger,\n    ),\n)\n\n',
  source: 'out/python/logger/client.py',
  blocks: {
    rootlogger: {
      start: 2,
      stop: 18,
    },
  },
  highlights: {},
};

export default snippet;
