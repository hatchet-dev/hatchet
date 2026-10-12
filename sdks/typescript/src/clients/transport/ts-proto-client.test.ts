import { create } from '@bufbuild/protobuf';
import { Code, ConnectError, type Transport } from '@connectrpc/connect';
import { Metadata } from 'nice-grpc-common';
import { EventsServiceDefinition } from '@hatchet/protoc/events/events';
import { EventsService } from '@hatchet/protoc-es/events/events_pb';
import {
  createTsProtoClient,
  headersToMetadata,
  metadataToHeaders,
  toConnectCallOptions,
} from './ts-proto-client';

type UnaryArgs = Parameters<Transport['unary']>;

interface Recorded {
  method: string;
  signal: AbortSignal | undefined;
  timeoutMs: number | undefined;
  header: Headers;
}

/**
 * An in-memory transport that records what each call carried and answers with the given
 * response headers and trailers. With `stall` set it never answers on its own; it only rejects
 * with Connect's cancellation error once the call's signal aborts, the way a real transport does
 * for a peer that never responds.
 */
function recordingTransport(options: { stall?: boolean; header?: Headers; trailer?: Headers }) {
  const calls: Recorded[] = [];
  const unary = (...args: UnaryArgs) => {
    const [method, signal, timeoutMs, header] = args;
    calls.push({ method: method.name, signal, timeoutMs, header: new Headers(header) });

    const response = {
      stream: false as const,
      service: method.parent,
      method,
      header: options.header ?? new Headers(),
      trailer: options.trailer ?? new Headers(),
      message: create(method.output),
    };

    if (!options.stall) {
      return Promise.resolve(response);
    }
    return new Promise((_, reject) => {
      signal?.addEventListener('abort', () =>
        reject(new ConnectError('This operation was aborted', Code.Canceled))
      );
    });
  };
  const transport: Transport = {
    unary: unary as Transport['unary'],
    stream: () => {
      throw new Error('unexpected stream');
    },
  };
  return { transport, calls };
}

describe('createTsProtoClient', () => {
  it('forwards the signal and converts a deadline to the remaining time', async () => {
    const { transport, calls } = recordingTransport({});
    const client = createTsProtoClient(EventsServiceDefinition, EventsService, transport);
    const controller = new AbortController();

    await client.putLog({}, { signal: controller.signal, deadline: Date.now() + 5_000 });
    await client.putLog({}, { deadline: new Date(Date.now() + 2_000) });

    expect(calls[0].signal).toBe(controller.signal);
    expect(calls[0].timeoutMs).toBeGreaterThan(4_000);
    expect(calls[0].timeoutMs).toBeLessThanOrEqual(5_000);
    expect(calls[1].timeoutMs).toBeGreaterThan(1_000);
    expect(calls[1].timeoutMs).toBeLessThanOrEqual(2_000);
  });

  it('sends request metadata as headers, base64 encoding binary values', async () => {
    const { transport, calls } = recordingTransport({});
    const client = createTsProtoClient(EventsServiceDefinition, EventsService, transport);

    await client.putLog(
      {},
      { metadata: Metadata({ 'x-request-id': 'abc', 'x-trace-bin': Uint8Array.of(1, 2, 3) }) }
    );

    expect(calls[0].header.get('x-request-id')).toBe('abc');
    expect(calls[0].header.get('x-trace-bin')).toBe(Buffer.from([1, 2, 3]).toString('base64'));
  });

  it('hands response headers and trailers to onHeader and onTrailer as metadata', async () => {
    const { transport } = recordingTransport({
      header: new Headers({
        'x-server': 'engine',
        'x-id-bin': Buffer.from('id').toString('base64'),
      }),
      trailer: new Headers({ 'grpc-status': '0' }),
    });
    const client = createTsProtoClient(EventsServiceDefinition, EventsService, transport);
    const onHeader = jest.fn();
    const onTrailer = jest.fn();

    await client.putLog({}, { onHeader, onTrailer });

    const [[header]]: Metadata[][] = onHeader.mock.calls;
    const [[trailer]]: Metadata[][] = onTrailer.mock.calls;
    expect(header.get('x-server')).toBe('engine');
    expect(header.get('x-id-bin')).toEqual(new Uint8Array(Buffer.from('id')));
    expect(trailer.get('grpc-status')).toBe('0');
  });

  it('rejects an already-aborted signal with an AbortError before sending', async () => {
    const { transport, calls } = recordingTransport({});
    const client = createTsProtoClient(EventsServiceDefinition, EventsService, transport);
    const controller = new AbortController();
    controller.abort();

    await expect(client.putLog({}, { signal: controller.signal })).rejects.toMatchObject({
      name: 'AbortError',
    });
    expect(calls).toHaveLength(0);
  });

  it('settles a stalled call with an AbortError when its signal aborts', async () => {
    const { transport } = recordingTransport({ stall: true });
    const client = createTsProtoClient(EventsServiceDefinition, EventsService, transport);
    const controller = new AbortController();

    const pending = client.putLog({}, { signal: controller.signal });
    setTimeout(() => controller.abort(), 10);

    await expect(pending).rejects.toMatchObject({ name: 'AbortError' });
  });

  it('rejects an expired deadline with DEADLINE_EXCEEDED before sending', async () => {
    const { transport, calls } = recordingTransport({});
    const client = createTsProtoClient(EventsServiceDefinition, EventsService, transport);

    await expect(client.putLog({}, { deadline: Date.now() - 1 })).rejects.toMatchObject({
      code: Code.DeadlineExceeded,
    });
    expect(calls).toHaveLength(0);
  });

  it('exposes every method of the definition', () => {
    const { transport } = recordingTransport({});
    const client = createTsProtoClient(EventsServiceDefinition, EventsService, transport);

    for (const name of Object.keys(EventsServiceDefinition.methods)) {
      expect(typeof (client as Record<string, unknown>)[name]).toBe('function');
    }
  });
});

describe('call option conversion', () => {
  it('returns undefined when no options are given', () => {
    expect(toConnectCallOptions(undefined)).toBeUndefined();
  });

  it('round-trips metadata through headers', () => {
    const metadata = Metadata({ 'x-a': 'one', 'x-b-bin': Uint8Array.of(9, 8) });

    const back = headersToMetadata(metadataToHeaders(metadata));

    expect(back.get('x-a')).toBe('one');
    expect(back.get('x-b-bin')).toEqual(Uint8Array.of(9, 8));
  });
});
