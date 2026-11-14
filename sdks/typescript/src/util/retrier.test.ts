import { createServer, type Http2Server, type Http2Session, type ServerHttp2Stream } from 'http2';
import type { AddressInfo } from 'net';
import { Code, ConnectError } from '@connectrpc/connect';
import { findTrailerError } from '@connectrpc/connect/protocol-grpc';
import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { createEventsRpc } from '@clients/event/rpc';
import { createNodeTransport } from '@clients/transport';
import { MAX_SERVER_RETRY_DELAY, retrier } from './retrier';
import sleep from './sleep';

jest.mock('./sleep', () => ({ __esModule: true, default: jest.fn() }));

const sleepMock = jest.mocked(sleep);
const realSleep: typeof sleep = jest.requireActual('./sleep').default;
const logger = DEFAULT_LOGGER('retrier-test', 'OFF');

/*
 * `grpc-status-details-bin` trailers exactly as grpc-go sends them for
 * status.New(codes.ResourceExhausted, "throttled").WithDetails(detail), with a RetryInfo of
 * 300 ms, a RetryInfo of 10 minutes and an ErrorInfo as the detail.
 */
const RETRY_INFO_300MS =
  'CAgSCXRocm90dGxlZBo0Cih0eXBlLmdvb2dsZWFwaXMuY29tL2dvb2dsZS5ycGMuUmV0cnlJbmZvEggKBhCAxoaPAQ';
const RETRY_INFO_10MIN =
  'CAgSCXRocm90dGxlZBoxCih0eXBlLmdvb2dsZWFwaXMuY29tL2dvb2dsZS5ycGMuUmV0cnlJbmZvEgUKAwjYBA';
const ERROR_INFO =
  'CAgSCXRocm90dGxlZBovCih0eXBlLmdvb2dsZWFwaXMuY29tL2dvb2dsZS5ycGMuRXJyb3JJbmZvEgMKAXg';

function trailers(code: Code, details?: string): Record<string, string> {
  return {
    'grpc-status': String(code),
    'grpc-message': 'throttled',
    ...(details ? { 'grpc-status-details-bin': details } : {}),
  };
}

/** The error the gRPC transport raises for a response with these trailers. */
function grpcError(code: Code, details?: string): ConnectError {
  const error = findTrailerError(new Headers(trailers(code, details)));
  if (!error) throw new Error('trailers carry no error');
  return error;
}

/** A call that fails with the given errors in order, then succeeds. */
function failing(...errors: unknown[]) {
  const fn = jest.fn(async () => 'ok');
  for (const error of errors) {
    fn.mockRejectedValueOnce(error);
  }
  return fn;
}

const waits = () => sleepMock.mock.calls.map(([ms]) => ms);

describe('retrier', () => {
  beforeEach(() => {
    sleepMock.mockResolvedValue(undefined);
    jest.spyOn(Math, 'random').mockReturnValue(0.5);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('backs off on its own schedule for errors without RetryInfo', async () => {
    const fn = failing(
      grpcError(Code.ResourceExhausted),
      grpcError(Code.Unavailable),
      grpcError(Code.ResourceExhausted, ERROR_INFO),
      new Error('boom')
    );

    await expect(retrier(fn, logger)).resolves.toBe('ok');
    expect(waits()).toEqual([150, 250, 450, 850]);
  });

  it('waits the RetryInfo delay of the failed attempt only', async () => {
    const fn = failing(
      grpcError(Code.ResourceExhausted, RETRY_INFO_300MS),
      grpcError(Code.Unavailable),
      grpcError(Code.ResourceExhausted, RETRY_INFO_300MS)
    );

    await expect(retrier(fn, logger)).resolves.toBe('ok');
    expect(waits()).toEqual([300, 250, 300]);
  });

  it('caps the RetryInfo delay', async () => {
    const fn = failing(grpcError(Code.ResourceExhausted, RETRY_INFO_10MIN));

    await expect(retrier(fn, logger)).resolves.toBe('ok');
    expect(waits()).toEqual([MAX_SERVER_RETRY_DELAY]);
  });

  it('keeps the attempt limit', async () => {
    const error = grpcError(Code.ResourceExhausted, RETRY_INFO_300MS);
    const fn = jest.fn(async () => {
      throw error;
    });

    await expect(retrier(fn, logger, { maxAttempts: 3 })).rejects.toBe(error);
    expect(fn).toHaveBeenCalledTimes(3);
    expect(waits()).toEqual([300, 300]);
  });

  it('does not retry an error shouldRetry refuses, even with RetryInfo', async () => {
    const error = grpcError(Code.ResourceExhausted, RETRY_INFO_300MS);
    const fn = failing(error);

    await expect(retrier(fn, logger, { shouldRetry: () => false })).rejects.toBe(error);
    expect(fn).toHaveBeenCalledTimes(1);
    expect(waits()).toEqual([]);
  });

  describe('over the gRPC transport', () => {
    let server: Http2Server;
    let sessions: Http2Session[];
    let attempts: number[];
    let responses: Array<Record<string, string>>;

    beforeEach(async () => {
      sleepMock.mockImplementation(realSleep);
      attempts = [];
      responses = [];
      sessions = [];
      server = createServer();
      server.on('session', (session) => sessions.push(session));
      server.on('stream', (stream) => {
        attempts.push(performance.now());
        stream.resume();
        stream.on('end', () => respond(stream, responses.shift()));
      });
      await new Promise<void>((resolve) => {
        server.listen(0, '127.0.0.1', resolve);
      });
    });

    // The client transport keeps its HTTP/2 session open and has no way to close it, and
    // server.close() waits for every session to end, so the sessions are destroyed first.
    afterEach(async () => {
      for (const session of sessions) {
        session.destroy();
      }
      await new Promise((resolve) => {
        server.close(resolve);
      });
    });

    /** Answers with the given error trailers (trailers-only), or with an empty message. */
    function respond(stream: ServerHttp2Stream, errorTrailers?: Record<string, string>) {
      const headers = { ':status': 200, 'content-type': 'application/grpc' };
      if (errorTrailers) {
        stream.respond({ ...headers, ...errorTrailers }, { endStream: true });
        return;
      }
      stream.respond(headers, { waitForTrailers: true });
      stream.on('wantTrailers', () => stream.sendTrailers({ 'grpc-status': '0' }));
      stream.end(Buffer.alloc(5));
    }

    it('waits the delay a throttled call asks for, then succeeds', async () => {
      const { port } = server.address() as AddressInfo;
      const events = createEventsRpc(
        createNodeTransport({
          token: 'test-token',
          host_port: `127.0.0.1:${port}`,
          tls_config: { tls_strategy: 'none' },
          api_url: 'http://127.0.0.1:1',
          tenant_id: 'tenant',
          log_level: 'OFF',
          logger: DEFAULT_LOGGER,
        })
      );
      responses.push(trailers(Code.ResourceExhausted, RETRY_INFO_300MS));

      await expect(
        retrier(() => events.push({ key: 'event', payload: '{}' }), logger)
      ).resolves.toBeDefined();

      expect(attempts).toHaveLength(2);
      const gap = attempts[1] - attempts[0];
      expect(gap).toBeGreaterThanOrEqual(300);
      expect(gap).toBeLessThan(2000);
    });
  });
});
