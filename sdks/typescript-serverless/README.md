# @hatchet-dev/serverless

The Hatchet serverless endpoint package. It is not published yet: for now this directory is the
generation target for the serverless wire contract, so the TypeScript side is produced from the
same protobuf definitions as the Go operator.

## Generated code

`src/generated/proto/` is ts-proto output for `api-contracts/v1/serverless.proto` and the
messages it imports (`v1/dispatcher.proto`, `v1/workflows.proto`, `v1/shared/*.proto` and the
package-less `dispatcher.proto`). The operator in `pkg/serverlessoperator` encodes the same
messages with protojson, so `fromJSON` and `toJSON` on the generated types read and write
exactly what the operator sends and expects.

Regenerate after changing any of those files:

```sh
pnpm install
pnpm run generate-proto
```

CI regenerates this directory (`task generate-serverless-proto`, part of `task generate-all`)
and fails when the committed output is stale.
