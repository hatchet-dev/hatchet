import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'unknown',
  content:
    "# Hatchet First Workflow Example\n\nThis is an example project demonstrating how to use Hatchet with Go. For detailed setup instructions, see the [Hatchet Setup Guide](https://docs.hatchet.run/home/setup).\n\n## Prerequisites\n\nBefore running this project, make sure you have the following:\n\n1. [Go v1.22 or higher](https://go.dev/doc/install)\n\n## Setup\n\n1. Clone the repository:\n\n```bash\ngit clone https://github.com/hatchet-dev/hatchet-go-quickstart.git\ncd hatchet-go-quickstart\n```\n\n2. Set the required environment variable `HATCHET_CLIENT_TOKEN` created in the [Getting Started Guide](https://docs.hatchet.run/home/hatchet-cloud-quickstart).\n\n```bash\nexport HATCHET_CLIENT_TOKEN=<token>\n```\n\n> Note: If you're self hosting you may need to set `HATCHET_CLIENT_TLS_STRATEGY=none` to disable TLS\n\n3. Install the project dependencies:\n\n```bash\ngo mod tidy\n```\n\n### Running an example\n\n1. Start a Hatchet worker:\n\n```bash\ngo run cmd/worker/main.go\n```\n\n2. In a new terminal, run the example task:\n\n```bash\ngo run cmd/run/main.go\n```\n\nThis will trigger the task on the worker running in the first terminal and print the output to the second terminal.\n",
  source: 'out/go/quickstart/README.md',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
