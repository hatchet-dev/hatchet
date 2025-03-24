import { CreateWorkflowVersionOpts, WorkflowVersion } from '@hatchet/protoc/workflows';
import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { ClientConfig } from '@clients/hatchet-client';
import { AdminClient } from './admin-client';
import { mockChannel, mockFactory } from '../hatchet-client/hatchet-client.test';
import { RunListenerClient } from '../listeners/run-listener/child-listener-client';

describe('AdminClient', () => {
  let client: AdminClient;

  it('should create a client', () => {
    const config: ClientConfig = {
      token:
        'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
      host_port: 'HOST_PORT',
      tls_config: {
        cert_file: 'TLS_CERT_FILE',
        key_file: 'TLS_KEY_FILE',
        ca_file: 'TLS_ROOT_CA_FILE',
        server_name: 'TLS_SERVER_NAME',
      },
      api_url: 'API_URL',
      tenant_id: 'tenantId',
      logger: DEFAULT_LOGGER,
    };

    const x = new AdminClient(
      config,
      mockChannel,
      mockFactory,
      {} as any,
      'tenantId',
      new RunListenerClient(config, mockChannel, mockFactory, {} as any)
    );

    expect(x).toBeDefined();
  });

  beforeEach(() => {
    const config: ClientConfig = {
      token:
        'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
      host_port: 'HOST_PORT',
      tls_config: {
        cert_file: 'TLS_CERT_FILE',
        key_file: 'TLS_KEY_FILE',
        ca_file: 'TLS_ROOT_CA_FILE',
        server_name: 'TLS_SERVER_NAME',
      },
      api_url: 'API_URL',
      tenant_id: 'tenantId',
      logger: DEFAULT_LOGGER,
    };

    client = new AdminClient(
      config,
      mockChannel,
      mockFactory,
      {} as any,
      'tenantId',
      new RunListenerClient(config, mockChannel, mockFactory, {} as any)
    );
  });

  describe('putWorkflow', () => {
    it('should throw an error if no version and not auto version', async () => {
      const workflow: CreateWorkflowVersionOpts = {
        name: 'workflow1',
        version: '',
        description: 'description1',
        eventTriggers: [],
        cronTriggers: [],
        scheduledTriggers: [],
        jobs: [],
        concurrency: undefined,
      };

      expect(() => client.putWorkflow(workflow)).rejects.toThrow(
        'PutWorkflow error: workflow version is required, or use autoVersion'
      );
    });

    it('should attempt to put the workflow', async () => {
      const workflow: CreateWorkflowVersionOpts = {
        name: 'workflow1',
        version: 'v0.0.1',
        description: 'description1',
        eventTriggers: [],
        cronTriggers: [],
        scheduledTriggers: [],
        jobs: [],
        concurrency: undefined,
      };

      const putSpy = jest.spyOn(client.client, 'putWorkflow').mockResolvedValue({
        id: 'workflow1',
        version: 'v0.1.0',
        order: 1,
        workflowId: 'workflow1',
        createdAt: undefined,
        updatedAt: undefined,
        scheduledWorkflows: [],
      });

      await client.putWorkflow(workflow);

      expect(putSpy).toHaveBeenCalled();
    });
  });

  describe('schedule_workflow', () => {
    it('should schedule a workflow', () => {
      const res: WorkflowVersion = {
        id: 'string',
        version: 'v0.0.1',
        order: 1,
        workflowId: 'string',
        scheduledWorkflows: [],
        createdAt: undefined,
        updatedAt: undefined,
      };

      const spy = jest.spyOn(client.client, 'scheduleWorkflow').mockResolvedValue(res);

      const now = new Date();

      client.scheduleWorkflow('workflowName', {
        schedules: [now],
      });

      expect(spy).toHaveBeenCalledWith({
        name: 'workflowName',
        schedules: [now],
      });
    });
  });
});
