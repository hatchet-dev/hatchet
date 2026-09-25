import { create, toBinary } from '@bufbuild/protobuf';
import { Code, ConnectError } from '@connectrpc/connect';
import { GetRunDetailsResponseSchema, RunStatus } from '@hatchet/protoc-es/v1/workflows_pb';
import { createV1AdminRpc } from '@hatchet/clients/admin/rpc';
import { createEventsRpc } from '@hatchet/clients/event/rpc';
import { createFetchTransport, resolveServerUrl } from './fetch-transport';
import { DEFAULT_MAX_MESSAGE_BYTES } from './transport';

const TOKEN = 'eyJhbGciOi.eyJzdWIi.sig';
const PROTO = { 'content-type': 'application/proto' };

/** A `GetRunDetails` response whose encoding is about `bytes` long. */
function detailsBody(bytes: number): Uint8Array {
  return toBinary(
    GetRunDetailsResponseSchema,
    create(GetRunDetailsResponseSchema, {
      status: RunStatus.COMPLETED,
      done: true,
      input: new Uint8Array(bytes - 16),
    })
  );
}

/** A stub fetch that answers every call with the given response and records what it saw. */
function stubFetch(respond: () => Response) {
  const requests: Array<{ url: string; bytes: number }> = [];
  const fetch: typeof globalThis.fetch = async (input, init) => {
    const body = init?.body as Uint8Array | undefined;
    requests.push({ url: String(input), bytes: body?.byteLength ?? 0 });
    return respond();
  };
  return { fetch, requests };
}

async function rejection(promise: Promise<unknown>): Promise<ConnectError> {
  try {
    await promise;
  } catch (e) {
    return ConnectError.from(e);
  }
  throw new Error('expected the call to reject');
}

describe('createFetchTransport message size limits', () => {
  it('decodes a response under the receive limit', async () => {
    const body = detailsBody(DEFAULT_MAX_MESSAGE_BYTES - 64);
    const { fetch } = stubFetch(() => new Response(body, { headers: PROTO }));
    const rpc = createV1AdminRpc(createFetchTransport({ token: TOKEN, serverUrl: 'https://e', fetch }));

    const details = await rpc.getRunDetails({ externalId: 'run' });

    expect(details.done).toBe(true);
    expect(details.input.byteLength).toBe(DEFAULT_MAX_MESSAGE_BYTES - 80);
  });

  it('rejects a response over 4 MiB with ResourceExhausted instead of decoding it', async () => {
    const body = detailsBody(DEFAULT_MAX_MESSAGE_BYTES + 16);
    const { fetch } = stubFetch(() => new Response(body, { headers: PROTO }));
    const rpc = createV1AdminRpc(createFetchTransport({ token: TOKEN, serverUrl: 'https://e', fetch }));

    const error = await rejection(rpc.getRunDetails({ externalId: 'run' }));

    expect(error.code).toBe(Code.ResourceExhausted);
    expect(error.rawMessage).toMatch(/larger than configured maxReceiveMessageBytes 4194304/);
  });

  it('cancels a streaming body once it passes the limit, counting decompressed bytes', async () => {
    const chunk = new Uint8Array(64 * 1024);
    let delivered = 0;
    let cancelled = false;
    // An endless body, as a compressed response expanding past the limit would be after
    // the runtime inflated it.
    const endless = () =>
      new ReadableStream<Uint8Array>({
        pull(controller) {
          delivered += chunk.byteLength;
          controller.enqueue(chunk);
        },
        cancel() {
          cancelled = true;
        },
      });
    const { fetch } = stubFetch(() => new Response(endless(), { headers: PROTO }));
    const rpc = createV1AdminRpc(createFetchTransport({ token: TOKEN, serverUrl: 'https://e', fetch }));

    const error = await rejection(rpc.getRunDetails({ externalId: 'run' }));

    expect(error.code).toBe(Code.ResourceExhausted);
    expect(cancelled).toBe(true);
    expect(delivered).toBeLessThanOrEqual(DEFAULT_MAX_MESSAGE_BYTES + 2 * chunk.byteLength);
  });

  it('holds a Connect error body to the same limit', async () => {
    const json = `{"code":"unknown","message":"${'x'.repeat(DEFAULT_MAX_MESSAGE_BYTES)}"}`;
    const { fetch } = stubFetch(
      () => new Response(json, { status: 500, headers: { 'content-type': 'application/json' } })
    );
    const rpc = createV1AdminRpc(createFetchTransport({ token: TOKEN, serverUrl: 'https://e', fetch }));

    const error = await rejection(rpc.getRunDetails({ externalId: 'run' }));

    expect(error.code).toBe(Code.ResourceExhausted);
  });

  it('refuses a request over 4 MiB before sending it', async () => {
    const { fetch, requests } = stubFetch(() => new Response(new Uint8Array(), { headers: PROTO }));
    const rpc = createEventsRpc(createFetchTransport({ token: TOKEN, serverUrl: 'https://e', fetch }));

    const error = await rejection(
      rpc.putLog({ taskRunExternalId: 'task', message: 'x'.repeat(DEFAULT_MAX_MESSAGE_BYTES) })
    );

    expect(error.code).toBe(Code.ResourceExhausted);
    expect(error.rawMessage).toMatch(/larger than configured maxSendMessageBytes 4194304/);
    expect(requests).toHaveLength(0);
  });

  it('sends a request under the limit and takes configured limits', async () => {
    const { fetch, requests } = stubFetch(() => new Response(new Uint8Array(), { headers: PROTO }));
    const rpc = createEventsRpc(
      createFetchTransport({
        token: TOKEN,
        serverUrl: 'https://e',
        fetch,
        maxSendMessageBytes: 100,
        maxReceiveMessageBytes: 8,
      })
    );

    await rpc.putLog({ taskRunExternalId: 'task', message: 'short' });
    expect(requests).toHaveLength(1);
    expect(requests[0].bytes).toBeLessThanOrEqual(100);

    const tooLong = await rejection(
      rpc.putLog({ taskRunExternalId: 'task', message: 'x'.repeat(100) })
    );
    expect(tooLong.code).toBe(Code.ResourceExhausted);
    expect(tooLong.rawMessage).toMatch(/maxSendMessageBytes 100/);

    const { fetch: bigAnswer } = stubFetch(
      () => new Response(new Uint8Array(9), { headers: PROTO })
    );
    const small = createEventsRpc(
      createFetchTransport({
        token: TOKEN,
        serverUrl: 'https://e',
        fetch: bigAnswer,
        maxReceiveMessageBytes: 8,
      })
    );
    const tooBig = await rejection(small.putLog({ taskRunExternalId: 'task', message: 'hi' }));
    expect(tooBig.code).toBe(Code.ResourceExhausted);
    expect(tooBig.rawMessage).toMatch(/maxReceiveMessageBytes 8/);
  });
});

describe('resolveServerUrl', () => {
  it.each([
    ['https://e.example', undefined, 'https://e.example'],
    ['https://e.example/', { strategy: 'tls' as const }, 'https://e.example'],
    ['HTTPS://E.example:7070/base///', undefined, 'HTTPS://E.example:7070/base'],
    ['http://e.example:7070', { strategy: 'none' as const }, 'http://e.example:7070'],
    ['http://127.0.0.1:7070/base', { strategy: 'none' as const }, 'http://127.0.0.1:7070/base'],
    ['http://[::1]:7070', { strategy: 'none' as const }, 'http://[::1]:7070'],
  ])('accepts serverUrl %s with tls %j', (serverUrl, tls, expected) => {
    expect(resolveServerUrl({ serverUrl, tls })).toBe(expected);
  });

  it.each([
    ['http://e.example:7070', undefined, /serverUrl is http:\/\/ but tls\.strategy is 'tls'/],
    ['http://e.example:7070', { strategy: 'tls' as const }, /serverUrl is http:\/\/ but tls\.strategy/],
    ['https://e.example', { strategy: 'none' as const }, /serverUrl is https:\/\/ but tls\.strategy is 'none'/],
    ['https://user:secret@e.example', undefined, /serverUrl must not carry a username or password/],
    ['https://e.example/base?x=1', undefined, /serverUrl must not carry a query string or fragment/],
    ['https://e.example/base?', undefined, /serverUrl must not carry a query string or fragment/],
    ['https://e.example/base#frag', undefined, /serverUrl must not carry a query string or fragment/],
    ['ftp://e.example', undefined, /serverUrl must be an absolute http:\/\/ or https:\/\/ URL/],
    ['e.example:7070', undefined, /serverUrl must be an absolute http:\/\/ or https:\/\/ URL/],
    ['//e.example', undefined, /serverUrl must be an absolute http:\/\/ or https:\/\/ URL/],
  ])('refuses serverUrl %s with tls %j', (serverUrl, tls, message) => {
    expect(() => resolveServerUrl({ serverUrl, tls })).toThrow(message);
    expect(() => createFetchTransport({ token: TOKEN, serverUrl, tls })).toThrow(message);
  });

  it('never echoes the URL, so a password in it stays out of the error', () => {
    expect(() => resolveServerUrl({ serverUrl: 'https://user:secret@e.example' })).toThrow(
      expect.not.objectContaining({ message: expect.stringContaining('secret') })
    );
  });

  it.each([
    ['e.example:7070', undefined, 'https://e.example:7070'],
    ['e.example:7070', { strategy: 'tls' as const }, 'https://e.example:7070'],
    ['e.example:7070', { strategy: 'none' as const }, 'http://e.example:7070'],
    ['dns:///e.example', undefined, 'https://e.example:443'],
  ])('derives the scheme for hostPort %s from tls %j', (hostPort, tls, expected) => {
    expect(resolveServerUrl({ hostPort, tls })).toBe(expected);
  });

  it('needs one of serverUrl and hostPort', () => {
    expect(() => resolveServerUrl({})).toThrow(/needs a serverUrl or a hostPort/);
  });
});
