import { LegacyHatchetClient } from '@clients/hatchet-client';
import { StepActionEventType, ActionType, AssignedAction } from '@hatchet/protoc/dispatcher';
import { ActionListener } from '@clients/dispatcher/action-listener';
import { never } from 'zod';
import sleep from '@util/sleep';
import { ChannelCredentials } from 'nice-grpc';
import { V0Worker } from './worker';

type AssignActionMock = AssignedAction | Error;

const mockStart: AssignActionMock = {
  tenantId: 'TENANT_ID',
  jobId: 'job1',
  jobName: 'Job One',
  jobRunId: 'run1',
  stepId: 'step1',
  stepRunId: 'runStep1',
  actionId: 'action1',
  actionType: ActionType.START_STEP_RUN,
  actionPayload: JSON.stringify('{"input": {"data": 1}}'),
  workflowRunId: 'workflowRun1',
  getGroupKeyRunId: 'groupKeyRun1',
  stepName: 'step1',
  retryCount: 0,
  priority: 1,
};

const mockCancel: AssignActionMock = {
  ...mockStart,
  actionType: ActionType.CANCEL_STEP_RUN,
};

describe('Worker', () => {
  let hatchet: LegacyHatchetClient;

  beforeEach(() => {
    hatchet = new LegacyHatchetClient(
      {
        token:
          'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
        log_level: 'OFF',
        host_port: 'HOST_PORT',
        tls_config: {
          cert_file: 'TLS_CERT_FILE',
          key_file: 'TLS_KEY_FILE',
          ca_file: 'TLS_ROOT_CA_FILE',
          server_name: 'TLS_SERVER_NAME',
        },
      },
      {
        credentials: ChannelCredentials.createInsecure(),
      }
    );
  });

  describe('registerWorkflow', () => {
    it('should update the registry', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });
      const putWorkflowSpy = jest.spyOn(worker.client.admin, 'putWorkflow').mockResolvedValue({
        id: 'workflow1',
        version: 'v0.1.0',
        order: 1,
        workflowId: 'workflow1',
        scheduledWorkflows: [],
        createdAt: undefined,
        updatedAt: undefined,
      });

      const workflow = {
        id: 'workflow1',
        description: 'test',
        on: {
          event: 'user:create',
        },
        steps: [
          {
            name: 'step1',
            run: (ctx: any) => {
              return { test: 'test' };
            },
          },
        ],
      };

      await worker.registerWorkflow(workflow);

      expect(putWorkflowSpy).toHaveBeenCalledTimes(1);

      expect(worker.action_registry).toEqual({
        [`workflow1:step1`]: workflow.steps[0].run,
      });
    });
  });

  describe('handle_start_step_run', () => {
    it('should start a step run', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });

      const getActionEventSpy = jest.spyOn(worker, 'getStepActionEvent');

      const sendActionEventSpy = jest
        .spyOn(worker.client.dispatcher, 'sendStepActionEvent')
        .mockResolvedValue({
          tenantId: 'TENANT_ID',
          workerId: 'WORKER_ID',
        });

      const startSpy = jest.fn().mockReturnValue({ data: 4 });

      worker.action_registry = {
        [mockStart.actionId]: startSpy,
      };

      worker.handleStartStepRun(mockStart);
      await sleep(100);

      expect(startSpy).toHaveBeenCalledTimes(1);

      expect(getActionEventSpy).toHaveBeenNthCalledWith(
        2,
        expect.anything(),
        StepActionEventType.STEP_EVENT_TYPE_COMPLETED,
        false,
        { data: 4 },
        0
      );
      expect(worker.futures[mockStart.stepRunId]).toBeUndefined();
      expect(sendActionEventSpy).toHaveBeenCalledTimes(2);
    });

    it('should fail gracefully', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });

      const getActionEventSpy = jest.spyOn(worker, 'getStepActionEvent');

      const sendActionEventSpy = jest
        .spyOn(worker.client.dispatcher, 'sendStepActionEvent')
        .mockResolvedValue({
          tenantId: 'TENANT_ID',
          workerId: 'WORKER_ID',
        });

      const startSpy = jest.fn().mockRejectedValue(new Error('ERROR'));

      worker.action_registry = {
        [mockStart.actionId]: startSpy,
      };

      worker.handleStartStepRun(mockStart);
      await sleep(100);

      expect(startSpy).toHaveBeenCalledTimes(1);
      expect(getActionEventSpy).toHaveBeenNthCalledWith(
        2,
        expect.anything(),
        StepActionEventType.STEP_EVENT_TYPE_FAILED,
        false,
        expect.anything(),
        0
      );
      expect(worker.futures[mockStart.stepRunId]).toBeUndefined();
      expect(sendActionEventSpy).toHaveBeenCalledTimes(2);
    });
  });

  describe('handle_cancel_step_run', () => {});

  describe('exit_gracefully', () => {
    xit('should call exit_gracefully on SIGTERM', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });

      // the spy is not working and the test is killing the test process
      const exitSpy = jest.spyOn(worker, 'exitGracefully').mockImplementationOnce(() => {
        throw new Error('Simulated error');
      });

      process.emit('SIGTERM', 'SIGTERM');
      expect(exitSpy).toHaveBeenCalledTimes(1);
    });

    xit('should call exit_gracefully on SIGINT', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });

      // This is killing the process (as it should) fix the spy at some point
      const exitSpy = jest.spyOn(worker, 'exitGracefully').mockResolvedValue();

      process.emit('SIGINT', 'SIGINT');
      expect(exitSpy).toHaveBeenCalledTimes(1);
    });

    xit('should unregister the listener and exit', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });

      jest.spyOn(process, 'exit').mockImplementation((number) => {
        throw new Error(`EXIT ${number}`);
      }); // This is killing the process (as it should) fix the spy at some point

      const mockActionListener = new ActionListener(hatchet.dispatcher, 'WORKER_ID');

      mockActionListener.unregister = jest.fn().mockResolvedValue(never());
      worker.listener = mockActionListener;

      expect(async () => {
        await worker.exitGracefully(true);
      }).toThrow('EXIT 0');
      expect(mockActionListener.unregister).toHaveBeenCalledTimes(1);
    });

    it('should exit the process if handle_kill is true', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });
      const exitSpy = jest.spyOn(process, 'exit').mockReturnValue(undefined as never);
      await worker.exitGracefully(true);
      expect(exitSpy).toHaveBeenCalledTimes(1);
    });
  });

  describe('start', () => {
    xit('should get actions and start runs', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });

      const startSpy = jest.spyOn(worker, 'handleStartStepRun').mockResolvedValue();
      const cancelSpy = jest.spyOn(worker, 'handleCancelStepRun').mockResolvedValue();

      const mockActionListener = new ActionListener(hatchet.dispatcher, 'WORKER_ID');

      const getActionListenerSpy = jest
        .spyOn(worker.client.dispatcher, 'getActionListener')
        .mockResolvedValue(mockActionListener);

      await worker.start();

      expect(getActionListenerSpy).toHaveBeenCalledTimes(1);
      expect(startSpy).toHaveBeenCalledTimes(2);
      expect(cancelSpy).toHaveBeenCalledTimes(0);
    });

    // it('should get actions and cancel runs', async () => {
    //   const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

    //   const startSpy = jest.spyOn(worker, 'handleStartStepRun').mockReturnValue();
    //   const cancelSpy = jest.spyOn(worker, 'handleCancelStepRun').mockReturnValue();

    //   const mockActionListener = new ActionListener(
    //     hatchet.dispatcher,
    //     mockListener([mockStart, mockCancel, new ServerError(Status.CANCELLED, 'CANCELLED')]),
    //     'WORKER_ID'
    //   );

    //   const getActionListenerSpy = jest
    //     .spyOn(worker.client.dispatcher, 'getActionListener')
    //     .mockResolvedValue(mockActionListener);

    //   await worker.start();

    //   expect(getActionListenerSpy).toHaveBeenCalledTimes(1);
    //   expect(startSpy).toHaveBeenCalledTimes(1);
    //   expect(cancelSpy).toHaveBeenCalledTimes(1);
    // });

    // it('should retry 5 times to start a worker then throw an error', async () => {
    //   const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

    //   const startSpy = jest.spyOn(worker, 'handleStartStepRun').mockReturnValue();
    //   const cancelSpy = jest.spyOn(worker, 'handleCancelStepRun').mockReturnValue();

    //   const mockActionListner = new ActionListener(
    //     hatchet.dispatcher,
    //     mockListener([mockStart, mockCancel, new ServerError(Status.CANCELLED, 'CANCELLED')]),
    //     'WORKER_ID'
    //   );

    //   const getActionListenerSpy = jest
    //     .spyOn(worker.client.dispatcher, 'getActionListener')
    //     .mockImplementationOnce(() => {
    //       throw new Error('Simulated error');
    //     })
    //     .mockImplementationOnce(() => {
    //       throw new Error('Simulated error');
    //     })
    //     .mockImplementationOnce(() => {
    //       throw new Error('Simulated error');
    //     })
    //     .mockImplementationOnce(() => {
    //       throw new Error('Simulated error');
    //     })
    //     .mockImplementationOnce(() => {
    //       throw new Error('Simulated error');
    //     })
    //     .mockImplementationOnce(() => {
    //       throw new Error('Simulated error');
    //     });

    //   await worker.start();

    //   expect(getActionListenerSpy).toHaveBeenCalledTimes(5);
    //   expect(startSpy).toHaveBeenCalledTimes(0);
    //   expect(cancelSpy).toHaveBeenCalledTimes(0);
    // });

    //   it('should successfully run after retrying < 5 times', async () => {
    //     const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

    //     const startSpy = jest.spyOn(worker, 'handleStartStepRun').mockReturnValue();
    //     const cancelSpy = jest.spyOn(worker, 'handleCancelStepRun').mockReturnValue();

    //     const mockActionLister = new ActionListener(
    //       hatchet.dispatcher,
    //       mockListener([mockStart, mockCancel, new ServerError(Status.CANCELLED, 'CANCELLED')]),
    //       'WORKER_ID'
    //     );

    //     const getActionListenerSpy = jest
    //       .spyOn(worker.client.dispatcher, 'getActionListener')
    //       .mockImplementationOnce(() => {
    //         throw new Error('Simulated error');
    //       })
    //       .mockImplementationOnce(() => {
    //         throw new Error('Simulated error');
    //       })
    //       .mockImplementationOnce(() => {
    //         throw new Error('Simulated error');
    //       })
    //       .mockImplementationOnce(() => {
    //         throw new Error('Simulated error');
    //       })
    //       .mockImplementationOnce(async () => mockActionLister);

    //     await worker.start();

    //     expect(getActionListenerSpy).toHaveBeenCalledTimes(5);
    //     expect(startSpy).toHaveBeenCalledTimes(1);
    //     expect(cancelSpy).toHaveBeenCalledTimes(1);
    //   });
  });
});
