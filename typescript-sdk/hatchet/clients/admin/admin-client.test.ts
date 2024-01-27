import { CreateWorkflowVersionOpts, WorkflowVersion } from '@protoc/workflows';
import { AdminClient } from './admin-client';
import { mockChannel, mockFactory } from '../hatchet-client/hatchet-client.test';

describe('AdminClient', () => {
  let client: AdminClient;

  it('should create a client', () => {
    const x = new AdminClient(
      {
        token: 'TOKEN',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      },
      mockChannel,
      mockFactory
    );

    expect(x).toBeDefined();
  });

  beforeEach(() => {
    client = new AdminClient(
      {
        token: 'TOKEN',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      },
      mockChannel,
      mockFactory
    );
  });

  describe('put_workflow', () => {
    it('should throw an error if no version and not auto version', async () => {
      const workflow: CreateWorkflowVersionOpts = {
        name: 'workflow1',
        version: '',
        description: 'description1',
        eventTriggers: [],
        cronTriggers: [],
        scheduledTriggers: [],
        jobs: [],
      };

      expect(() => client.put_workflow(workflow, { autoVersion: false })).rejects.toThrow(
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
      };

      const putSpy = jest.spyOn(client.client, 'putWorkflow').mockResolvedValue({
        id: 'workflow1',
        version: 'v0.1.0',
        order: 1,
        workflowId: 'workflow1',
        jobs: [],
        createdAt: undefined,
        updatedAt: undefined,
        triggers: undefined,
      });

      await client.put_workflow(workflow);

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
        jobs: [],
        createdAt: undefined,
        updatedAt: undefined,
        triggers: undefined,
      };

      const spy = jest.spyOn(client.client, 'scheduleWorkflow').mockResolvedValue(res);

      const now = new Date();

      client.schedule_workflow('workflowId', {
        schedules: [now],
      });

      expect(spy).toHaveBeenCalledWith({
        workflowId: 'workflowId',
        schedules: [now],
      });
    });
  });
});
