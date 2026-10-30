import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import {
  Dispatcher,
  type RefreshTimeoutRequest,
} from '@hatchet/protoc-es/dispatcher/dispatcher_pb';
import { HatchetCore } from './client';

const TENANT_ID = '707d0855-80ab-4e1f-a156-f1c4546cbf52';
const TASK_RUN_ID = '4e9f0f2e-5d6c-4a1f-9f0e-3b1c4d5e6f72';
const TIMEOUT_AT = new Date('2026-10-22T12:00:00Z');

function makeToken(claims: Record<string, unknown>): string {
  const encode = (value: unknown) => Buffer.from(JSON.stringify(value)).toString('base64url');
  return `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode(claims)}.signature`;
}

const TOKEN = makeToken({ sub: TENANT_ID, grpc_broadcast_address: 'engine.example.com:7070' });

/** An in-memory dispatcher: records the task-side calls and fails them on request. */
function fakeDispatcher() {
  const refreshes: RefreshTimeoutRequest[] = [];
  const releases: string[] = [];
  let failure: ConnectError | undefined;

  const transport = createRouterTransport(({ service }) => {
    service(Dispatcher, {
      refreshTimeout: (req) => {
        if (failure) throw failure;
        refreshes.push(req);
        return { timeoutAt: timestampFromDate(TIMEOUT_AT) };
      },
      releaseSlot: (req) => {
        if (failure) throw failure;
        releases.push(req.taskRunExternalId);
        return {};
      },
    });
  });

  return {
    transport,
    refreshes,
    releases,
    failWith(error: ConnectError) {
      failure = error;
    },
  };
}

describe('core tasks', () => {
  it('refreshTimeout sends the task run id and the increment as a Go duration', async () => {
    const dispatcher = fakeDispatcher();
    const client = new HatchetCore({ token: TOKEN, transport: dispatcher.transport });

    const timeoutAt = await client.tasks.refreshTimeout(TASK_RUN_ID, { seconds: 15 });
    await client.tasks.refreshTimeout(TASK_RUN_ID, 90_000);
    await client.tasks.refreshTimeout(TASK_RUN_ID, '2h');

    expect(timeoutAt).toEqual(TIMEOUT_AT);
    expect(dispatcher.refreshes.map((r) => r.taskRunExternalId)).toEqual([
      TASK_RUN_ID,
      TASK_RUN_ID,
      TASK_RUN_ID,
    ]);
    expect(dispatcher.refreshes.map((r) => r.incrementTimeoutBy)).toEqual(['15s', '1m30s', '2h']);
  });

  it('releaseSlot sends the task run id', async () => {
    const dispatcher = fakeDispatcher();
    const client = new HatchetCore({ token: TOKEN, transport: dispatcher.transport });

    await expect(client.tasks.releaseSlot(TASK_RUN_ID)).resolves.toBeUndefined();

    expect(dispatcher.releases).toEqual([TASK_RUN_ID]);
  });

  it("rejects with the engine's error", async () => {
    const dispatcher = fakeDispatcher();
    const client = new HatchetCore({ token: TOKEN, transport: dispatcher.transport });
    dispatcher.failWith(new ConnectError('task run not found', Code.NotFound));

    await expect(client.tasks.refreshTimeout(TASK_RUN_ID, '10s')).rejects.toMatchObject({
      code: Code.NotFound,
    });
    await expect(client.tasks.releaseSlot(TASK_RUN_ID)).rejects.toMatchObject({
      code: Code.NotFound,
    });
  });

  it('honors an aborted signal before sending', async () => {
    const dispatcher = fakeDispatcher();
    const client = new HatchetCore({ token: TOKEN, transport: dispatcher.transport });
    const controller = new AbortController();
    controller.abort();

    await expect(
      client.tasks.releaseSlot(TASK_RUN_ID, { signal: controller.signal })
    ).rejects.toMatchObject({ name: 'AbortError' });
    expect(dispatcher.releases).toHaveLength(0);
  });
});
