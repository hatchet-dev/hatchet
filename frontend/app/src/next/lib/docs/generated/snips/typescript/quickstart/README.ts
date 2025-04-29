import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'unknown',
  'content': '# Hatchet First Workflow Example\n\nThis is an example project demonstrating how to use Hatchet with TypeScript. For detailed setup instructions, see the [Hatchet Setup Guide](https://docs.hatchet.run/home/setup).\n\n## Prerequisites\n\nBefore running this project, make sure you have the following:\n\n1. [Node.js v16 or higher](https://nodejs.org/en/download)\n2. npm, yarn, or pnpm package manager\n\n## Setup\n\n1. Clone the repository:\n\n```bash\ngit clone https://github.com/hatchet-dev/hatchet-typescript-quickstart.git\ncd hatchet-typescript-quickstart\n```\n\n2. Set the required environment variable `HATCHET_CLIENT_TOKEN` created in the [Getting Started Guide](https://docs.hatchet.run/home/hatchet-cloud-quickstart).\n\n```bash\nexport HATCHET_CLIENT_TOKEN=<token>\n```\n\n> Note: If you\'re self hosting you may need to set `HATCHET_CLIENT_TLS_STRATEGY=none` to disable TLS\n\n3. Install the project dependencies:\n\n```bash\nnpm install\n# or\nyarn install\n# or\npnpm install\n```\n\n### Running an example\n\n1. Start a Hatchet worker:\n\n```bash\nnpm run start\n```\n\n2. In a new terminal, run the example task:\n\n```bash\nnpm run run:simple\n```\n\nThis will trigger the task on the worker running in the first terminal and print the output to the second terminal.\n',
  'source': 'out/typescript/quickstart/README.md',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
