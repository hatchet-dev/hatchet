/**
 * The healthcheck body: every workflow in full, the action ids served here, whether the
 * durable relay is available, and the runtime. The operator registers the workflows under
 * the names declared here, the way a worker does, when the canonical form changes.
 */
import { CreateWorkflowVersionRequest as SdkCreateWorkflowVersionRequest } from '@hatchet-dev/typescript-sdk/edge/index.js';
import { CreateWorkflowVersionRequest } from '../generated/proto/v1/workflows';
import { ServerlessHealthcheckResponse } from '../generated/proto/v1/serverless';
import { version as sdkVersion } from '../../package.json';
import type { Registry } from './registry';

export interface HealthcheckRuntime {
  /** The platform, such as "cloudflare-workers" or "vercel". */
  name: string;
}

export function buildHealthcheck(
  registry: Registry,
  runtime: HealthcheckRuntime,
  durableSupported: boolean
): ServerlessHealthcheckResponse {
  return {
    // The SDK's generated CreateWorkflowVersionRequest and the package's are two copies
    // of the same message. Passing through protojson keeps them apart as types and is
    // exactly what the operator reads.
    workflows: registry.workflows.map((workflow) =>
      CreateWorkflowVersionRequest.fromJSON(SdkCreateWorkflowVersionRequest.toJSON(workflow.proto))
    ),
    actions: [...registry.served].sort(),
    durable: { supported: durableSupported },
    // The served non-durable tasks the operator invokes over a socket, so they can await
    // child runs on it. Durable tasks have a socket regardless and are not listed.
    tasks: [...registry.streamActions]
      .filter((action) => registry.served.has(action))
      .sort()
      .map((action) => ({ action, streams: true })),
    runtime: { name: runtime.name, sdkVersion },
  };
}

export function serializeHealthcheck(response: ServerlessHealthcheckResponse): string {
  return JSON.stringify(ServerlessHealthcheckResponse.toJSON(response));
}
