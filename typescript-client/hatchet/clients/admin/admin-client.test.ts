import { CreateWorkflowVersionOpts, Workflow, WorkflowVersion } from '@protoc/workflows';
import { ServerError, Status } from 'nice-grpc-common';
import { AdminClient } from './admin-client';
import { mockChannel } from '../hatchet-client/hatchet-client.test';

describe('AdminClient', () => {
  let client: AdminClient;

  it('should create a client', () => {
    const x = new AdminClient(
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
    client = new AdminClient(
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

  describe('should_put', () => {
    let workflow: CreateWorkflowVersionOpts;

    beforeAll(() => {
      workflow = {
        name: 'workflow1',
        version: '',
        description: 'description1',
        eventTriggers: [],
        cronTriggers: [],
        scheduledTriggers: [],
        jobs: [],
      };
    });

    it('shouldPut:F and vX; if auto:F', () => {
      const existing: any = {
        versions: [
          {
            version: 'v0.0.1',
          },
        ],
      };
      const [shouldPut, version] = AdminClient.should_put(
        {
          ...workflow,
          version: 'vX',
        },
        existing,
        {
          autoVersion: false,
        }
      );
      expect(shouldPut).toEqual(false);
      expect(version).toEqual('vX');
    });

    it('shouldPut:T and v0.0.1 if auto:T, no version, and no Existing', () => {
      const existing: any = undefined;
      const [shouldPut, version] = AdminClient.should_put(
        {
          ...workflow,
          version: '',
        },
        existing,
        {
          autoVersion: true,
        }
      );
      expect(shouldPut).toEqual(true);
      expect(version).toEqual('v0.1.0');
    });

    it('shouldPut:T and bump version if auto:T, no version, and has Existing', () => {
      const existing: any = {
        versions: [
          {
            version: 'v0.0.1',
          },
        ],
      };
      const [shouldPut, version] = AdminClient.should_put(
        {
          ...workflow,
          version: '',
        },
        existing,
        {
          autoVersion: true,
        }
      );

      expect(shouldPut).toEqual(true);
      expect(version).toEqual('v0.1.0');
    });

    it('shouldPut:T and keep existing version if auto:T, with a version, and no Existing', () => {
      const existing: any = {
        versions: [],
      };
      const [shouldPut, version] = AdminClient.should_put(
        {
          ...workflow,
          version: 'vKEEP',
        },
        existing,
        {
          autoVersion: true,
        }
      );

      expect(shouldPut).toEqual(true);
      expect(version).toEqual('vKEEP');
    });
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

    it('should check if an existing workflow exists, if not it should put', async () => {
      const workflow: CreateWorkflowVersionOpts = {
        name: 'workflow1',
        version: 'v0.0.1',
        description: 'description1',
        eventTriggers: [],
        cronTriggers: [],
        scheduledTriggers: [],
        jobs: [],
      };

      const existingSpy = jest
        .spyOn(client.client, 'getWorkflowByName')
        .mockRejectedValue(new ServerError(Status.NOT_FOUND, 'not found'));

      const shouldPutSpy = jest.spyOn(AdminClient, 'should_put').mockReturnValue([true, 'v0.1.0']);

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

      expect(existingSpy).toHaveBeenCalledWith({
        tenantId: 'TENANT_ID',
        name: 'workflow1',
      });

      expect(shouldPutSpy).toHaveBeenCalledWith(workflow, undefined, undefined);

      expect(putSpy).toHaveBeenCalled();
    });

    it('should check if an existing workflow exists, if not it should put', async () => {
      const workflow: CreateWorkflowVersionOpts = {
        name: 'workflow1',
        version: 'v0.0.1',
        description: 'description1',
        eventTriggers: [],
        cronTriggers: [],
        scheduledTriggers: [],
        jobs: [],
      };

      const mockExisting: Workflow = {
        id: 'workflow1',
        name: 'workflow1',
        tenantId: 'TENANT_ID',
        description: 'description1',
        versions: [
          {
            id: 'workflow1',
            version: 'v0.0.1',
            order: 1,
            workflowId: 'workflow1',
            jobs: [],
            createdAt: undefined,
            updatedAt: undefined,
            triggers: undefined,
          },
        ],
        createdAt: undefined,
        updatedAt: undefined,
      };

      const existingSpy = jest
        .spyOn(client.client, 'getWorkflowByName')
        .mockResolvedValue(mockExisting);

      const shouldPutSpy = jest.spyOn(AdminClient, 'should_put').mockReturnValue([false, 'v0.2.0']);

      const putSpy = jest.spyOn(client.client, 'putWorkflow').mockResolvedValue({
        id: 'workflow1',
        version: 'v0.2.0',
        order: 1,
        workflowId: 'workflow1',
        jobs: [],
        createdAt: undefined,
        updatedAt: undefined,
        triggers: undefined,
      });

      await client.put_workflow(workflow, { autoVersion: false });

      expect(existingSpy).toHaveBeenCalledWith({
        tenantId: 'TENANT_ID',
        name: 'workflow1',
      });

      expect(shouldPutSpy).toHaveBeenCalledWith(workflow, mockExisting, { autoVersion: false });

      expect(putSpy).not.toHaveBeenCalled();
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
        tenantId: 'TENANT_ID',
        workflowId: 'workflowId',
        schedules: [now],
      });
    });
  });
});
