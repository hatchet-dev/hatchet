## YAML Workflow Example

This example shows how you can create a YAML file in your repository to define the structure of a workflow. This example runs the [sample-workflow.yaml](./.hatchet/sample-workflow.yaml).

## Explanation

This folder contains a demo example of a workflow that simply echoes the input message as an output. The workflow file showcases the following features:

- Running a simple job with a set of dependent steps
- Variable references within step arguments -- each subsequent step in a workflow can call `.steps.<step_id>.<field>` to access output arguments

## How to run

Navigate to this directory and run the following steps:

1. Make sure you have a Hatchet server running (see the instructions [here](../../README.md)). After running `task seed`, grab the tenant ID which is output to the console.
2. Set your environment variables -- if you're using the bundled Temporal server, this will look like:

```sh
cat > .env <<EOF
HATCHET_CLIENT_TENANT_ID=<tenant-id-from-seed-command>
HATCHET_CLIENT_TLS_ROOT_CA_FILE=../../hack/dev/certs/ca.cert
HATCHET_CLIENT_TLS_CERT_FILE=../../hack/dev/certs/client-worker.pem
HATCHET_CLIENT_TLS_KEY_FILE=../../hack/dev/certs/client-worker.key
HATCHET_CLIENT_TLS_SERVER_NAME=cluster
EOF
```

3. Run the following within this directory:

```sh
go run main.go';
```
