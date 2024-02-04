import { HatchetClient } from '@clients/hatchet-client';
import { ActionEventType, ActionType, AssignedAction } from '@hatchet/protoc/dispatcher';
import { ActionListener } from '@clients/dispatcher/action-listener';
import { ServerError, Status } from 'nice-grpc-common';
import { mockListener } from '@clients/dispatcher/action-listener.test';
import { never } from 'zod';
import sleep from '@util/sleep';
import { ChannelCredentials } from 'nice-grpc';
import { Worker } from './worker';
import { Context } from '@hatchet/step';

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
};

const mockCancel: AssignActionMock = {
  ...mockStart,
  actionType: ActionType.CANCEL_STEP_RUN,
};

describe('Worker', () => {
  let hatchet: HatchetClient;

  beforeEach(() => {
    hatchet = new HatchetClient(
      {
        token: 'TOKEN',
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

  describe('register_workflow', () => {
    it('should update the registry', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });
      const putWorkflowSpy = jest.spyOn(worker.client.admin, 'put_workflow').mockResolvedValue();

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

      await worker.register_workflow(workflow);

      expect(putWorkflowSpy).toHaveBeenCalledTimes(1);

      expect(worker.action_registry).toEqual({
        [`default:step1`]: workflow.steps[0].run,
      });
    });
  });

  describe('handle_start_step_run', () => {
    it('should start a step run', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

      const putWorkflowSpy = jest.spyOn(worker.client.admin, 'put_workflow').mockResolvedValue();

      const getActionEventSpy = jest.spyOn(worker, 'get_action_event');

      const sendActionEventSpy = jest
        .spyOn(worker.client.dispatcher, 'send_action_event')
        .mockResolvedValue({
          tenantId: 'TENANT_ID',
          workerId: 'WORKER_ID',
        });

      const startSpy = jest.fn().mockReturnValue({ data: 4 });

      worker.action_registry = {
        [mockStart.actionId]: startSpy,
      };

      worker.handle_start_step_run(mockStart);
      await sleep(100);

      expect(startSpy).toHaveBeenCalledTimes(1);

      expect(getActionEventSpy).toHaveBeenNthCalledWith(
        2,
        expect.anything(),
        ActionEventType.STEP_EVENT_TYPE_COMPLETED,
        { data: 4 }
      );
      expect(worker.futures[mockStart.stepRunId]).toBeUndefined();
      expect(sendActionEventSpy).toHaveBeenCalledTimes(2);
    });

    it('should fail gracefully', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

      const getActionEventSpy = jest.spyOn(worker, 'get_action_event');

      const sendActionEventSpy = jest
        .spyOn(worker.client.dispatcher, 'send_action_event')
        .mockResolvedValue({
          tenantId: 'TENANT_ID',
          workerId: 'WORKER_ID',
        });

      const startSpy = jest.fn().mockRejectedValue(new Error('ERROR'));

      worker.action_registry = {
        [mockStart.actionId]: startSpy,
      };

      worker.handle_start_step_run(mockStart);
      await sleep(100);

      expect(startSpy).toHaveBeenCalledTimes(1);
      expect(getActionEventSpy).toHaveBeenNthCalledWith(
        2,
        expect.anything(),
        ActionEventType.STEP_EVENT_TYPE_FAILED,
        expect.anything()
      );
      expect(worker.futures[mockStart.stepRunId]).toBeUndefined();
      expect(sendActionEventSpy).toHaveBeenCalledTimes(2);
    });
  });

  describe('handle_cancel_step_run', () => {
    it('should cancel a step run', () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

      const cancelSpy = jest.fn().mockReturnValue(undefined);

      worker.futures = {
        [mockCancel.stepRunId]: {
          cancel: cancelSpy,
        } as any,
      };

      worker.handle_cancel_step_run(mockCancel);

      expect(cancelSpy).toHaveBeenCalledTimes(1);
      expect(worker.futures[mockCancel.stepRunId]).toBeUndefined();
    });
  });

  describe('exit_gracefully', () => {
    xit('should call exit_gracefully on SIGTERM', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

      // the spy is not working and the test is killing the test process
      const exitSpy = jest.spyOn(worker, 'exit_gracefully').mockImplementationOnce(() => {
        throw new Error('Simulated error');
      });

      process.emit('SIGTERM', 'SIGTERM');
      expect(exitSpy).toHaveBeenCalledTimes(1);
    });

    xit('should call exit_gracefully on SIGINT', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

      // This is killing the process (as it should) fix the spy at some point
      const exitSpy = jest.spyOn(worker, 'exit_gracefully').mockResolvedValue();

      process.emit('SIGINT', 'SIGINT');
      expect(exitSpy).toHaveBeenCalledTimes(1);
    });

    xit('should unregister the listener and exit', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

      jest.spyOn(process, 'exit').mockImplementation((number) => {
        throw new Error(`EXIT ${number}`);
      }); // This is killing the process (as it should) fix the spy at some point

      const mockActionListener = new ActionListener(
        hatchet.dispatcher,
        mockListener([mockStart, mockStart, new ServerError(Status.CANCELLED, 'CANCELLED')]),
        'WORKER_ID'
      );

      mockActionListener.unregister = jest.fn().mockResolvedValue(never());
      worker.listener = mockActionListener;

      expect(async () => {
        await worker.exit_gracefully();
      }).toThrow('EXIT 0');
      expect(mockActionListener.unregister).toHaveBeenCalledTimes(1);
    });

    it('should exit the process if handle_kill is true', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });
      const exitSpy = jest.spyOn(process, 'exit').mockReturnValue(undefined as never);
      await worker.exit_gracefully();
      expect(exitSpy).toHaveBeenCalledTimes(1);
    });
  });

  describe('start', () => {
    xit('should get actions and start runs', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

      const startSpy = jest.spyOn(worker, 'handle_start_step_run').mockReturnValue();
      const cancelSpy = jest.spyOn(worker, 'handle_cancel_step_run').mockReturnValue();

      const mockActionListener = new ActionListener(
        hatchet.dispatcher,
        mockListener([mockStart, mockStart, new ServerError(Status.CANCELLED, 'CANCELLED')]),
        'WORKER_ID'
      );

      const getActionListenerSpy = jest
        .spyOn(worker.client.dispatcher, 'get_action_listener')
        .mockResolvedValue(mockActionListener);

      await worker.start();

      expect(getActionListenerSpy).toHaveBeenCalledTimes(1);
      expect(startSpy).toHaveBeenCalledTimes(2);
      expect(cancelSpy).toHaveBeenCalledTimes(0);
    });

    it('should get actions and cancel runs', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

      const startSpy = jest.spyOn(worker, 'handle_start_step_run').mockReturnValue();
      const cancelSpy = jest.spyOn(worker, 'handle_cancel_step_run').mockReturnValue();

      const mockActionListner = new ActionListener(
        hatchet.dispatcher,
        mockListener([mockStart, mockCancel, new ServerError(Status.CANCELLED, 'CANCELLED')]),
        'WORKER_ID'
      );

      const getActionListenerSpy = jest
        .spyOn(worker.client.dispatcher, 'get_action_listener')
        .mockResolvedValue(mockActionListner);

      await worker.start();

      expect(getActionListenerSpy).toHaveBeenCalledTimes(1);
      expect(startSpy).toHaveBeenCalledTimes(1);
      expect(cancelSpy).toHaveBeenCalledTimes(1);
    });

    it('should retry 5 times to start a worker then throw an error', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

      const startSpy = jest.spyOn(worker, 'handle_start_step_run').mockReturnValue();
      const cancelSpy = jest.spyOn(worker, 'handle_cancel_step_run').mockReturnValue();

      const mockActionListner = new ActionListener(
        hatchet.dispatcher,
        mockListener([mockStart, mockCancel, new ServerError(Status.CANCELLED, 'CANCELLED')]),
        'WORKER_ID'
      );

      const getActionListenerSpy = jest
        .spyOn(worker.client.dispatcher, 'get_action_listener')
        .mockImplementationOnce(() => {
          throw new Error('Simulated error');
        })
        .mockImplementationOnce(() => {
          throw new Error('Simulated error');
        })
        .mockImplementationOnce(() => {
          throw new Error('Simulated error');
        })
        .mockImplementationOnce(() => {
          throw new Error('Simulated error');
        })
        .mockImplementationOnce(() => {
          throw new Error('Simulated error');
        })
        .mockImplementationOnce(() => {
          throw new Error('Simulated error');
        });

      await worker.start();

      expect(getActionListenerSpy).toHaveBeenCalledTimes(5);
      expect(startSpy).toHaveBeenCalledTimes(0);
      expect(cancelSpy).toHaveBeenCalledTimes(0);
    });

    it('should successfully run after retrying < 5 times', async () => {
      const worker = new Worker(hatchet, { name: 'WORKER_NAME' });

      const startSpy = jest.spyOn(worker, 'handle_start_step_run').mockReturnValue();
      const cancelSpy = jest.spyOn(worker, 'handle_cancel_step_run').mockReturnValue();

      const mockActionLister = new ActionListener(
        hatchet.dispatcher,
        mockListener([mockStart, mockCancel, new ServerError(Status.CANCELLED, 'CANCELLED')]),
        'WORKER_ID'
      );

      const getActionListenerSpy = jest
        .spyOn(worker.client.dispatcher, 'get_action_listener')
        .mockImplementationOnce(() => {
          throw new Error('Simulated error');
        })
        .mockImplementationOnce(() => {
          throw new Error('Simulated error');
        })
        .mockImplementationOnce(() => {
          throw new Error('Simulated error');
        })
        .mockImplementationOnce(() => {
          throw new Error('Simulated error');
        })
        .mockImplementationOnce(async () => mockActionLister);

      await worker.start();

      expect(getActionListenerSpy).toHaveBeenCalledTimes(5);
      expect(startSpy).toHaveBeenCalledTimes(1);
      expect(cancelSpy).toHaveBeenCalledTimes(1);
    });
  });
});
