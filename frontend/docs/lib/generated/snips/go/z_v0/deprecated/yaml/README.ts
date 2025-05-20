import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "unknown",
  "content": "## YAML Workflow Example\n\nThis example shows how you can create a YAML file in your repository to define the structure of a workflow. This example runs the [sample-workflow.yaml](./.hatchet/sample-workflow.yaml).\n\n## Explanation\n\nThis folder contains a demo example of a workflow that simply echoes the input message as an output. The workflow file showcases the following features:\n\n- Running a simple job with a set of dependent steps\n- Variable references within step arguments -- each subsequent step in a workflow can call `.steps.<step_id>.<field>` to access output arguments\n\n## How to run\n\nNavigate to this directory and run the following steps:\n\n1. Make sure you have a Hatchet server running (see the instructions [here](../../README.md)). After running `task seed`, grab the tenant ID which is output to the console.\n2. Set your environment variables -- if you're using the bundled Temporal server, this will look like:\n\n```sh\ncat > .env <<EOF\nHATCHET_CLIENT_TENANT_ID=<tenant-id-from-seed-command>\nHATCHET_CLIENT_TLS_ROOT_CA_FILE=../../hack/dev/certs/ca.cert\nHATCHET_CLIENT_TLS_CERT_FILE=../../hack/dev/certs/client-worker.pem\nHATCHET_CLIENT_TLS_KEY_FILE=../../hack/dev/certs/client-worker.key\nHATCHET_CLIENT_TLS_SERVER_NAME=cluster\nEOF\n```\n\n3. Run the following within this directory:\n\n```sh\ngo run main.go';\n```\n",
  "source": "out/go/z_v0/deprecated/yaml/README.md",
  "blocks": {},
  "highlights": {}
};

export default snippet;
