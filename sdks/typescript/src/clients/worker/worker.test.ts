import { LegacyHatchetClient } from '@clients/hatchet-client';
import { StepActionEventType, ActionType, AssignedAction } from '@hatchet/protoc/dispatcher';
import { Action, ActionListener } from '@clients/dispatcher/action-listener';
import { never } from 'zod';
import sleep from '@util/sleep';
import { ChannelCredentials } from 'nice-grpc';
import { V0Worker } from './worker';

type AssignActionMock = AssignedAction | Error;

const mockStart: Action = {
  tenantId: 'TENANT_ID',
  jobId: 'job1',
  jobName: 'Job One',
  jobRunId: 'run1',
  taskId: 'step1',
  taskRunExternalId: 'runStep1',
  actionId: 'action1',
  actionType: ActionType.START_STEP_RUN,
  actionPayload: JSON.stringify('{"input": {"data": 1}}'),
  workflowRunId: 'workflowRun1',
  getGroupKeyRunId: 'groupKeyRun1',
  taskName: 'step1',
  retryCount: 0,
  priority: 1,
};

const mockCancel: Action = {
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
      expect(worker.futures[mockStart.taskRunExternalId]).toBeUndefined();
      expect(sendActionEventSpy).toHaveBeenCalledTimes(2);
    });

    it('should apply middleware before/after (merge semantics)', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });

      const order: string[] = [];
      const seenInputs: any[] = [];

      hatchet.config.middleware = {
        before: (_input: any, ctx: any) => {
          order.push('before');
          expect(ctx.taskRunExternalId()).toBe(mockStart.taskRunExternalId);
          return { data: 2 };
        },
        after: (_output: any, _ctx: any, input: any) => {
          order.push('after');
          return { observed: input.data };
        },
      };

      const getActionEventSpy = jest.spyOn(worker, 'getStepActionEvent');

      jest.spyOn(worker.client.dispatcher, 'sendStepActionEvent').mockResolvedValue({
        tenantId: 'TENANT_ID',
        workerId: 'WORKER_ID',
      });

      const startSpy = jest.fn().mockImplementation((ctx: any) => {
        order.push('step');
        seenInputs.push(ctx.input);
        return { ok: true };
      });

      worker.action_registry = {
        [mockStart.actionId]: startSpy,
      };

      worker.handleStartStepRun(mockStart);
      await sleep(100);

      expect(order).toEqual(['before', 'step', 'after']);
      // before merges { data: 2 } into existing input
      expect(seenInputs[0]).toEqual(expect.objectContaining({ data: 2 }));

      // after merges { observed: 2 } into the task result { ok: true }
      expect(getActionEventSpy).toHaveBeenNthCalledWith(
        2,
        expect.anything(),
        StepActionEventType.STEP_EVENT_TYPE_COMPLETED,
        false,
        { ok: true, observed: 2 },
        0
      );
    });

    it('should apply array of before/after hooks in order', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });

      const order: string[] = [];
      const seenInputs: any[] = [];

      hatchet.config.middleware = {
        before: [
          (_input: any, _ctx: any) => {
            order.push('before1');
            return { a: 1 };
          },
          (_input: any, _ctx: any) => {
            order.push('before2');
            return { b: 2 };
          },
        ],
        after: [
          (_output: any, _ctx: any, _input: any) => {
            order.push('after1');
            return { x: 10 };
          },
          (_output: any, _ctx: any, _input: any) => {
            order.push('after2');
            return { y: 20 };
          },
        ],
      };

      const getActionEventSpy = jest.spyOn(worker, 'getStepActionEvent');

      jest.spyOn(worker.client.dispatcher, 'sendStepActionEvent').mockResolvedValue({
        tenantId: 'TENANT_ID',
        workerId: 'WORKER_ID',
      });

      const startSpy = jest.fn().mockImplementation((ctx: any) => {
        order.push('step');
        seenInputs.push(ctx.input);
        return { ok: true };
      });

      worker.action_registry = {
        [mockStart.actionId]: startSpy,
      };

      worker.handleStartStepRun(mockStart);
      await sleep(100);

      expect(order).toEqual(['before1', 'before2', 'step', 'after1', 'after2']);
      expect(seenInputs[0]).toEqual(expect.objectContaining({ a: 1, b: 2 }));

      expect(getActionEventSpy).toHaveBeenNthCalledWith(
        2,
        expect.anything(),
        StepActionEventType.STEP_EVENT_TYPE_COMPLETED,
        false,
        { ok: true, x: 10, y: 20 },
        0
      );
    });

    it('should treat middleware errors as task errors', async () => {
      const worker = new V0Worker(hatchet, { name: 'WORKER_NAME' });

      hatchet.config.middleware = {
        before: () => {
          throw new Error('middleware exploded');
        },
      };

      const getActionEventSpy = jest.spyOn(worker, 'getStepActionEvent');

      jest.spyOn(worker.client.dispatcher, 'sendStepActionEvent').mockResolvedValue({
        tenantId: 'TENANT_ID',
        workerId: 'WORKER_ID',
      });

      const startSpy = jest.fn();

      worker.action_registry = {
        [mockStart.actionId]: startSpy,
      };

      worker.handleStartStepRun(mockStart);
      await sleep(100);

      expect(startSpy).not.toHaveBeenCalled();
      expect(getActionEventSpy).toHaveBeenNthCalledWith(
        2,
        expect.anything(),
        StepActionEventType.STEP_EVENT_TYPE_FAILED,
        false,
        expect.anything(),
        0
      );
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
      expect(worker.futures[mockStart.taskRunExternalId]).toBeUndefined();
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
