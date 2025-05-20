import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'unknown',
  content:
    'certs/\n\n# Environments\n.env\nenv/\nvenv/\n.venv/\n__pycache__/\n*.py[cod]\n*$py.class\n.Python\n.pytest_cache/\n.coverage\nhtmlcov/\n\n# Distribution / packaging\ndist/\nbuild/\n*.egg-info/\n*.egg\n\n.DS_Store\n\nindex/index.json\n',
  source: 'out/python/quickstart/.gitignore',
  blocks: {},
  highlights: {},
};

export default snippet;
