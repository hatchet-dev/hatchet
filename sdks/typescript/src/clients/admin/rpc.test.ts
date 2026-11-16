import { create } from '@bufbuild/protobuf';
import type { Transport } from '@connectrpc/connect';
import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import { WorkflowServiceDefinition } from '@hatchet/protoc/workflows/workflows';
import { AdminServiceDefinition } from '@hatchet/protoc/v1/workflows';
import { EventsServiceDefinition } from '@hatchet/protoc/events/events';
import { AdminClient as V1AdminClient } from '@hatchet/v1/client/admin';
import { EventClient } from '@clients/event/event-client';
import { RunListenerClient } from '@clients/listeners/run-listener/child-listener-client';
import { createChannel, createClientFactory } from 'nice-grpc';
import { AdminClient } from './admin-client';
import { createV1AdminRpc, createWorkflowsRpc } from './rpc';

const config: ClientConfig = {
  token: 'test-token',
  host_port: '127.0.0.1:1',
  tls_config: { tls_strategy: 'none' },
  api_url: 'http://127.0.0.1:1',
  tenant_id: 'tenant',
  log_level: 'OFF',
  logger: DEFAULT_LOGGER,
};

// The streaming channel the legacy constructors take; nothing here connects through it.
const mockChannel = createChannel(config.host_port);
const mockFactory = createClientFactory();

// Answers every unary call with an empty response without touching the network.
const transport: Transport = {
  unary: async (method) => ({
    stream: false,
    service: method.parent,
    method,
    header: new Headers(),
    trailer: new Headers(),
    message: create(method.output),
  }),
  stream: () => {
    throw new Error('unexpected stream');
  },
};

function methodNames(definition: { methods: Record<string, unknown> }): string[] {
  return Object.keys(definition.methods).sort();
}

function functionNames(client: object): string[] {
  return Object.keys(client)
    .filter((key) => typeof (client as Record<string, unknown>)[key] === 'function')
    .sort();
}

describe('public RPC client surface', () => {
  it('serves every WorkflowService and AdminService method on the v1 admin client', () => {
    const admin = new V1AdminClient(config, {} as never, {} as never, transport);

    expect(functionNames(admin.workflowsGrpc)).toEqual(methodNames(WorkflowServiceDefinition));
    expect(functionNames(admin.adminGrpc)).toEqual(methodNames(AdminServiceDefinition));
  });

  it('serves every WorkflowService and AdminService method on the legacy admin client', () => {
    const listener = new RunListenerClient(config, mockChannel, mockFactory, {} as never);
    const admin = new AdminClient(
      config,
      mockChannel,
      mockFactory,
      {} as never,
      config.tenant_id,
      listener,
      undefined,
      transport
    );

    expect(functionNames(admin.client)).toEqual(methodNames(WorkflowServiceDefinition));
    expect(functionNames(admin.v1Client)).toEqual(methodNames(AdminServiceDefinition));
  });

  it('serves every EventsService method on the event client', () => {
    const events = new EventClient(config, mockChannel, mockFactory, {} as never, transport);

    expect(functionNames(events.client)).toEqual(methodNames(EventsServiceDefinition));
  });

  it('answers the restored methods with the ts-proto response types', async () => {
    const admin = createV1AdminRpc(transport);
    const workflows = createWorkflowsRpc(transport);

    await expect(admin.cancelTasks({ externalIds: [] })).resolves.toEqual({
      cancelledTasks: [],
    });
    await expect(admin.getRunDetails({ externalId: 'run' })).resolves.toMatchObject({
      status: 0,
      taskRuns: {},
    });
    await expect(workflows.triggerWorkflow({ name: 'wf' })).resolves.toEqual({
      workflowRunId: '',
    });
  });
});
