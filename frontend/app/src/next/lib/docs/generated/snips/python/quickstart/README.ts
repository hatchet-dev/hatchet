import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'unknown',
  'content': '## Hatchet Python Quickstart\n\nThis is an example project demonstrating how to use Hatchet with Python. For detailed setup instructions, see the [Hatchet Setup Guide](https://docs.hatchet.run/home/setup).\n\n## Prerequisites\n\nBefore running this project, make sure you have the following:\n\n1. [Python v3.10 or higher](https://www.python.org/downloads/)\n2. [Poetry](https://python-poetry.org/docs/#installation) for dependency management\n\n## Setup\n\n1. Clone the repository:\n\n```bash\ngit clone https://github.com/hatchet-dev/hatchet-python-quickstart.git\ncd hatchet-python-quickstart\n```\n\n2. Set the required environment variable `HATCHET_CLIENT_TOKEN` created in the [Getting Started Guide](https://docs.hatchet.run/home/hatchet-cloud-quickstart).\n\n```bash\nexport HATCHET_CLIENT_TOKEN=<token>\n```\n\n> Note: If you\'re self hosting you may need to set `HATCHET_CLIENT_TLS_STRATEGY=none` to disable TLS\n\n3. Install the project dependencies:\n\n```bash\npoetry install\n```\n\n### Running an example\n\n1. Start a Hatchet worker by running the following command:\n\n```shell\npoetry run python src/worker.py\n```\n\n2. To run the example workflow, open a new terminal and run the following command:\n\n```shell\npoetry run python src/run.py\n```\n\nThis will trigger the workflow on the worker running in the first terminal and print the output to the the second terminal.\n',
  'source': 'out/python/quickstart/README.md',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
