import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'unknown',
  content:
    "[tool.poetry]\nname = 'hatchet-python-quickstart'\nversion = '0.1.0'\ndescription = 'Simple Setup to Run Hatchet Workflows'\nauthors = ['gabriel ruttner <gabe@hatchet.run>']\nreadme = 'README.md'\npackage-mode = false\n\n[tool.poetry.dependencies]\npython = '^3.10'\nhatchet-sdk = '1.0.0a1'\n\n\n[build-system]\nrequires = ['poetry-core']\nbuild-backend = 'poetry.core.masonry.api'\n\n[tool.poetry.scripts]\nsimple = 'src.run:main'\nworker = 'src.worker:main'\n",
  source: 'out/python/quickstart/pyproject.toml',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
