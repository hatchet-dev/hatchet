import { StepActionEventType } from '@hatchet/protoc/dispatcher';
import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { DispatcherClient } from './dispatcher-client';
import { mockChannel, mockFactory } from '../hatchet-client/hatchet-client.test';

let client: DispatcherClient;

describe('DispatcherClient', () => {
  it('should create a client', () => {
    const x = new DispatcherClient(
      {
        token:
          'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',

        host_port: 'HOST_PORT',
        log_level: 'OFF',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
        api_url: 'API_URL',
        tenant_id: 'tenantId',
        logger: DEFAULT_LOGGER,
      },
      mockChannel,
      mockFactory
    );

    expect(x).toBeDefined();
  });

  beforeEach(() => {
    client = new DispatcherClient(
      {
        token:
          'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',

        host_port: 'HOST_PORT',
        log_level: 'OFF',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
        api_url: 'API_URL',
        tenant_id: 'tenantId',
        logger: DEFAULT_LOGGER,
      },
      mockChannel,
      mockFactory
    );
  });

  describe('get_action_listener', () => {
    //   it('should register the worker', async () => {
    //     const clientSpy = jest.spyOn(client.client, 'register').mockResolvedValue({
    //       workerId: 'WORKER_ID',
    //       tenantId: 'TENANT_ID',
    //       workerName: 'WORKER_NAME',
    //     });
    //     const listenerSpy = jest.spyOn(client.client, 'listen');
    //     const listener = await client.getActionListener({
    //       workerName: 'WORKER_NAME',
    //       services: ['SERVICE'],
    //       actions: ['ACTION'],
    //     });
    //     expect(clientSpy).toHaveBeenCalledWith({
    //       workerName: 'WORKER_NAME',
    //       services: ['SERVICE'],
    //       actions: ['ACTION'],
    //     });
    //     expect(listenerSpy).toHaveBeenCalledWith({
    //       workerId: 'WORKER_ID',
    //     });
    //     expect(listener).toBeDefined();
    //     expect(listener.workerId).toEqual('WORKER_ID');
    //   });
  });

  describe('send_action_event', () => {
    it('should send action events', () => {
      const clientSpy = jest.spyOn(client.client, 'sendStepActionEvent').mockResolvedValue({
        tenantId: 'TENANT_ID',
        workerId: 'WORKER_ID',
      });

      client.sendStepActionEvent({
        workerId: 'WORKER_ID',
        actionId: 'ACTION_ID',
        eventType: StepActionEventType.STEP_EVENT_TYPE_COMPLETED,
        eventPayload: '{"foo":"bar"}',
        eventTimestamp: new Date(),
        jobId: 'a',
        jobRunId: 'b',
        stepId: 'c',
        stepRunId: 'd',
      });

      expect(clientSpy).toHaveBeenCalledWith({
        workerId: 'WORKER_ID',
        actionId: 'ACTION_ID',
        eventType: StepActionEventType.STEP_EVENT_TYPE_COMPLETED,
        eventPayload: '{"foo":"bar"}',
        jobId: 'a',
        jobRunId: 'b',
        stepId: 'c',
        stepRunId: 'd',
        eventTimestamp: expect.any(Object),
      });
    });
  });
});
