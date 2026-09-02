import { ActionType, AssignedAction } from '@hatchet/protoc/dispatcher';
import sleep from '@util/sleep';
import { AbortError } from '@hatchet/util/abort-error';
import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { DispatcherClient } from './dispatcher-client';
import { ActionListener } from './action-listener';
import { mockChannel, mockFactory } from '../../legacy/legacy-client.test';
import {
  LISTENER_RECONNECT_REASON,
  LISTENER_SHUTDOWN_REASON,
} from './listener-abort';

jest.mock('@util/sleep', () => ({
  __esModule: true,
  default: jest.fn(() => Promise.resolve()),
}));

let dispatcher: DispatcherClient;

type AssignActionMock = AssignedAction | Error;

// Mock data for AssignedAction
const mockAssignedActions: AssignActionMock[] = [
  {
    tenantId: 'tenant1',
    jobId: 'job1',
    jobName: 'Job One',
    jobRunId: 'run1',
    taskId: 'step1',
    taskRunExternalId: 'runStep1',
    actionId: 'action1',
    actionType: ActionType.START_STEP_RUN,
    actionPayload: 'payload1',
    workflowRunId: 'workflowRun1',
    getGroupKeyRunId: 'groupKeyRun1',
    taskName: 'step1',
    retryCount: 0,
    priority: 1,
  },
  // ... Add more mock AssignedAction objects as needed
];

// Mock implementation of the listener
export const mockListener = (fixture: AssignActionMock[]) =>
  (async function* gen() {
    for (const action of fixture) {
      // Simulate asynchronous behavior
      await sleep(100);

      if (action instanceof Error) {
        throw action;
      }

      yield action;
    }
  })();

describe('ActionListener', () => {
  beforeEach(() => {
    dispatcher = new DispatcherClient(
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

  it('should create a client', async () => {
    const listener = new ActionListener(dispatcher, 'WORKER_ID');
    expect(listener).toBeDefined();
    expect(listener.workerId).toEqual('WORKER_ID');
  });

  describe('actions', () => {
    // it('it should "yield" actions', async () => {
    //   const listener = new ActionListener(
    //     dispatcher,
    //     'WORKER_ID',
    //     100,
    //     5,
    //   );
    //   const retrySpy = jest.spyOn(listener, 'getListenClient');
    //   retrySpy.mockReturnValue(
    //     mockListener([...mockAssignedActions, new ServerError(Status.CANCELLED, 'CANCELLED')])
    //   );
    //   const actions = listener.actions();
    //   const res = [];
    //   for await (const action of actions) {
    //     res.push(action);
    //   }
    //   expect(res[0]).toEqual({
    //     tenantId: 'tenant1',
    //     jobId: 'job1',
    //     jobName: 'Job One',
    //     jobRunId: 'run1',
    //     taskId: 'step1',
    //     taskRunId: 'runStep1',
    //     actionId: 'action1',
    //     actionType: ActionType.START_STEP_RUN,
    //     actionPayload: 'payload1',
    //     workflowRunId: 'workflowRun1',
    //     getGroupKeyRunId: 'groupKeyRun1',
    //   });
    // });
    //   it('it should break on grpc CANCELLED', async () => {
    //     const listener = new ActionListener(
    //       dispatcher,
    //       mockListener([...mockAssignedActions, new ServerError(Status.CANCELLED, 'CANCELLED')]),
    //       'WORKER_ID'
    //     );
    //     const actions = listener.actions();
    //     // throw an error from listen client
    //     const retrySpy = jest.spyOn(listener, 'getListenClient');
    //     const res = [];
    //     for await (const action of actions) {
    //       res.push(action);
    //     }
    //     expect(res.length).toEqual(1);
    //     expect(retrySpy).toHaveBeenCalledTimes(1);
    //   });
    //   it('it should break on unknown error', async () => {
    //     const listener = new ActionListener(
    //       dispatcher,
    //       mockListener([...mockAssignedActions, new Error('Simulated error')]),
    //       'WORKER_ID'
    //     );
    //     const actions = listener.actions();
    //     const retrySpy = jest.spyOn(listener, 'getListenClient');
    //     const res = [];
    //     for await (const action of actions) {
    //       res.push(action);
    //     }
    //     expect(res.length).toEqual(1);
    //     expect(retrySpy).toHaveBeenCalledTimes(6);
    //   });
    //   it('it should attempt to re-establish connection on grpc UNAVAILABLE', async () => {
    //     const listener = new ActionListener(
    //       dispatcher,
    //       mockListener([...mockAssignedActions, new ServerError(Status.UNAVAILABLE, 'UNAVAILABLE')]),
    //       'WORKER_ID'
    //     );
    //     const retrySpy = jest.spyOn(listener, 'getListenClient');
    //     const actions = listener.actions();
    //     const res = [];
    //     for await (const action of actions) {
    //       res.push(action);
    //     }
    //     expect(res.length).toEqual(1);
    //     expect(retrySpy).toHaveBeenCalledTimes(6);
    //   });
    // });
    // describe('retry_subscribe', () => {
    //   it('should exit after successful connection', async () => {
    //     const listener = new ActionListener(
    //       dispatcher,
    //       mockListener(mockAssignedActions),
    //       'WORKER_ID'
    //     );
    //     // Mock the listener to throw an error on the first call
    //     const listenSpy = jest
    //       .spyOn(listener.client, 'listen')
    //       .mockReturnValue(mockListener(mockAssignedActions));
    //     await listener.getListenClient();
    //     expect(listenSpy).toHaveBeenCalledTimes(1);
    //   });
    //   it('should retry until success', async () => {
    //     const listener = new ActionListener(
    //       dispatcher,
    //       mockListener(mockAssignedActions),
    //       'WORKER_ID'
    //     );
    //     // Mock the listener to throw an error on the first call
    //     // const listenSpy = jest
    //     //   .spyOn(listener.client, 'listen')
    //     //   .mockImplementationOnce(() => {
    //     //     throw new Error('Simulated error');
    //     //   })
    //     //   .mockImplementationOnce(() => mockListener(mockAssignedActions));
    //     await expect(async () => {
    //       await listener.getListenClient();
    //     }).not.toThrow();
    //   });
    //   it('should not throw an error if successful', async () => {
    //     const listener = new ActionListener(
    //       dispatcher,
    //       mockListener(mockAssignedActions),
    //       'WORKER_ID'
    //     );
    //     // Mock the listener to throw an error on the first call
    //     const listenSpy = jest
    //       .spyOn(listener.client, 'listen')
    //       .mockImplementationOnce(() => {
    //         throw new Error('Simulated error');
    //       })
    //       .mockImplementationOnce(() => mockListener(mockAssignedActions));
    //     await listener.getListenClient();
    //     expect(listenSpy).toHaveBeenCalledTimes(2);
    //   });
    //   it('should retry at most COUNT times and throw an error', async () => {
    //     const listener = new ActionListener(
    //       dispatcher,
    //       mockListener(mockAssignedActions),
    //       'WORKER_ID'
    //     );
    //     // Mock the listener to throw an error on the first call
    //     const listenSpy = jest
    //       .spyOn(listener.client, 'listen')
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
    //       .mockImplementationOnce(() => {
    //         throw new Error('Simulated error');
    //       })
    //       .mockImplementationOnce(() => {
    //         throw new Error('Simulated error');
    //       })
    //       .mockImplementationOnce(() => mockListener(mockAssignedActions));
    //     try {
    //       await listener.getListenClient();
    //       expect(listenSpy).toHaveBeenCalledTimes(5);
    //     } catch (e: any) {
    //       expect(e.message).toEqual(`Could not subscribe to the worker after 5 retries`);
    //     }
    //   });
  });

  describe('unregister', () => {
    // it('should unsubscribe itself', async () => {
    //   const listener = new ActionListener(
    //     dispatcher,
    //     mockListener(mockAssignedActions),
    //     'WORKER_ID'
    //   );
    //   const unsubscribeSpy = jest.spyOn(listener.client, 'unsubscribe').mockResolvedValue({
    //     tenantId: 'TENANT_ID',
    //     workerId: 'WORKER_ID',
    //   });
    //   const res = await listener.unregister();
    //   expect(unsubscribeSpy).toHaveBeenCalled();
    //   expect(res.workerId).toEqual('WORKER_ID');
    // });
  });

  describe('inactive stream reconnect', () => {
    it('reconnects on listener reconnect abort without exiting the generator', async () => {
      const listener = new ActionListener(dispatcher, 'WORKER_ID', 1, 5);
      const getListenClientSpy = jest.spyOn(listener, 'getListenClient');

      getListenClientSpy
        .mockResolvedValueOnce(
          (async function* first() {
            yield mockAssignedActions[0] as AssignedAction;
            throw new AbortError(LISTENER_RECONNECT_REASON);
          })()
        )
        .mockResolvedValueOnce(
          (async function* second() {
            yield mockAssignedActions[0] as AssignedAction;
            throw new AbortError(LISTENER_SHUTDOWN_REASON);
          })()
        );

      const actions = listener.actions();
      const first = await actions.next();
      expect(first.done).toBe(false);

      const second = await actions.next();
      expect(second.done).toBe(false);

      expect(getListenClientSpy).toHaveBeenCalledTimes(2);
    });

    it('exits the generator on shutdown abort', async () => {
      const listener = new ActionListener(dispatcher, 'WORKER_ID', 1, 5);
      const getListenClientSpy = jest.spyOn(listener, 'getListenClient');

      getListenClientSpy.mockResolvedValueOnce(
        (async function* shutdown() {
          yield mockAssignedActions[0] as AssignedAction;
          throw new AbortError(LISTENER_SHUTDOWN_REASON);
        })()
      );

      const actions = listener.actions();
      const results = [];
      for await (const action of actions) {
        results.push(action);
      }

      expect(results).toHaveLength(1);
      expect(getListenClientSpy).toHaveBeenCalledTimes(1);
    });

    it('marks worker unhealthy when stream goes inactive', () => {
      const listener = new ActionListener(dispatcher, 'WORKER_ID');
      const statusChanges: string[] = [];
      listener.setWorkerStatusCallback((status) => statusChanges.push(status));

      (listener as any).handleStreamInactive();

      expect(statusChanges).toEqual(['UNHEALTHY']);
    });
  });
});
