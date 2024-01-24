import { ActionEventType } from '@protoc/dispatcher';
import { DispatcherClient } from './dispatcher-client';
import { mockChannel } from '../hatchet-client/hatchet-client.test';

let client: DispatcherClient;

describe('DispatcherClient', () => {
  it('should create a client', () => {
    const x = new DispatcherClient(
      {
        tenant_id: 'TENANT_ID',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      },
      mockChannel
    );

    expect(x).toBeDefined();
  });

  beforeEach(() => {
    client = new DispatcherClient(
      {
        tenant_id: 'TENANT_ID',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      },
      mockChannel
    );
  });

  describe('get_action_listener', () => {
    it('should register the worker', async () => {
      const clientSpy = jest.spyOn(client.client, 'register').mockResolvedValue({
        workerId: 'WORKER_ID',
        tenantId: 'TENANT_ID',
        workerName: 'WORKER_NAME',
      });

      const listenerSpy = jest.spyOn(client.client, 'listen');

      const listener = await client.get_action_listener({
        workerName: 'WORKER_NAME',
        services: ['SERVICE'],
        actions: ['ACTION'],
      });

      expect(clientSpy).toHaveBeenCalledWith({
        tenantId: 'TENANT_ID',
        workerName: 'WORKER_NAME',
        services: ['SERVICE'],
        actions: ['ACTION'],
      });

      expect(listenerSpy).toHaveBeenCalledWith({
        tenantId: 'TENANT_ID',
        workerId: 'WORKER_ID',
      });

      expect(listener).toBeDefined();
      expect(listener.workerId).toEqual('WORKER_ID');
    });
  });

  describe('send_action_event', () => {
    it('should send action events', () => {
      const clientSpy = jest.spyOn(client.client, 'sendActionEvent').mockResolvedValue({
        tenantId: 'TENANT_ID',
        workerId: 'WORKER_ID',
      });

      client.send_action_event({
        tenantId: 'TENANT_ID',
        workerId: 'WORKER_ID',
        actionId: 'ACTION_ID',
        eventType: ActionEventType.STEP_EVENT_TYPE_COMPLETED,
        eventPayload: '{"foo":"bar"}',
        eventTimestamp: new Date(),
        jobId: 'a',
        jobRunId: 'b',
        stepId: 'c',
        stepRunId: 'd',
      });

      expect(clientSpy).toHaveBeenCalledWith({
        tenantId: 'TENANT_ID',
        workerId: 'WORKER_ID',
        actionId: 'ACTION_ID',
        eventType: ActionEventType.STEP_EVENT_TYPE_COMPLETED,
        eventPayload: '{"foo":"bar"}',
        jobId: 'a',
        jobRunId: 'b',
        stepId: 'c',
        stepRunId: 'd',
        eventTimestamp: expect.any(Date),
      });
    });
  });
});
