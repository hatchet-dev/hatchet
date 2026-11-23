# SDK conformance scenarios

Behaviour that every Hatchet SDK must implement the same way, written once as data.

Each SDK's unit tests load these files and run every scenario. A change to one SDK that breaks the shared behaviour fails that SDK's tests, and a change to a scenario here runs the CI of all four SDKs, because each SDK workflow triggers on `api-contracts/**`.

Each file's `description` field explains its format. To change shared behaviour, change the scenario first and then every SDK until its tests pass.

This is a convention for now: nothing checks that an SDK runs a file. Each SDK keeps one test that loads it, listed below, and a new SDK needs to add its own.

## `throttled_calls.json`

How an SDK retries a unary gRPC call that the engine throttles. Run by:

- Go: `pkg/client/retry/grpc_test.go`
- Python: `sdks/python/tests/unit/test_server_retry_delay.py`
- TypeScript: `sdks/typescript/src/util/retrier.test.ts`
- Ruby: `sdks/ruby/src/spec/hatchet/clients/grpc/server_retry_delay_spec.rb`
