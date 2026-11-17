import { readFileSync } from 'fs';
import { join } from 'path';
import { BinaryWriter, WireType } from '@bufbuild/protobuf/wire';
import { Code, ConnectError } from '@connectrpc/connect';
import { findTrailerError } from '@connectrpc/connect/protocol-grpc';
import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { DEFAULT_RETRY_COUNT, MAX_THROTTLED_ATTEMPTS, retrier } from './retrier';
import sleep from './sleep';

jest.mock('./sleep', () => ({ __esModule: true, default: jest.fn() }));

const sleepMock = jest.mocked(sleep);
const logger = DEFAULT_LOGGER('retrier-test', 'OFF');

/** The throttled-call rule all Hatchet SDKs share; its `description` explains the format. */
const { scenarios } = JSON.parse(
  readFileSync(
    join(__dirname, '../../../../api-contracts/sdk-conformance/throttled_calls.json'),
    'utf8'
  )
) as {
  scenarios: { name: string; delays_ms: number[]; waits_ms: number[]; succeeds: boolean }[];
};

/*
 * `grpc-status-details-bin` trailers exactly as grpc-go sends them for
 * status.New(codes.ResourceExhausted, "throttled").WithDetails(detail), with a RetryInfo of
 * 300 ms and an ErrorInfo as the detail.
 */
const RETRY_INFO_300MS =
  'CAgSCXRocm90dGxlZBo0Cih0eXBlLmdvb2dsZWFwaXMuY29tL2dvb2dsZS5ycGMuUmV0cnlJbmZvEggKBhCAxoaPAQ';
const ERROR_INFO =
  'CAgSCXRocm90dGxlZBovCih0eXBlLmdvb2dsZWFwaXMuY29tL2dvb2dsZS5ycGMuRXJyb3JJbmZvEgMKAXg';

/**
 * The `grpc-status-details-bin` trailer for status.New(codes.ResourceExhausted, "throttled")
 * with a RetryInfo of `ms` as the detail, base64 without padding like the constants above.
 */
function retryInfo(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const nanos = (ms % 1000) * 1_000_000;
  const duration = new BinaryWriter();
  if (seconds) duration.tag(1, WireType.Varint).int64(seconds);
  if (nanos) duration.tag(2, WireType.Varint).int32(nanos);
  const status = new BinaryWriter()
    .tag(1, WireType.Varint)
    .int32(Code.ResourceExhausted)
    .tag(2, WireType.LengthDelimited)
    .string('throttled')
    .tag(3, WireType.LengthDelimited)
    .fork()
    .tag(1, WireType.LengthDelimited)
    .string('type.googleapis.com/google.rpc.RetryInfo')
    .tag(2, WireType.LengthDelimited)
    .fork()
    .tag(1, WireType.LengthDelimited)
    .bytes(duration.finish())
    .join()
    .join()
    .finish();
  return Buffer.from(status).toString('base64').replace(/=+$/, '');
}

/** The error the gRPC transport raises for a response with this status and details trailer. */
function grpcError(code: Code, details?: string): ConnectError {
  const error = findTrailerError(
    new Headers({
      'grpc-status': String(code),
      'grpc-message': 'throttled',
      ...(details ? { 'grpc-status-details-bin': details } : {}),
    })
  );
  if (!error) throw new Error('trailers carry no error');
  return error;
}

/** Runs the retrier over a call that fails with the given errors in order, then succeeds. */
async function waitsAfter(...errors: unknown[]): Promise<unknown[]> {
  const fn = jest.fn(async () => 'ok');
  for (const error of errors) {
    fn.mockRejectedValueOnce(error);
  }
  await expect(retrier(fn, logger)).resolves.toBe('ok');
  return sleepMock.mock.calls.map(([ms]) => ms);
}

describe('retrier', () => {
  beforeEach(() => {
    sleepMock.mockReset();
    jest.spyOn(Math, 'random').mockReturnValue(0.5);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('backs off on its own schedule after an error without a RetryInfo delay', async () => {
    const waits = await waitsAfter(
      grpcError(Code.ResourceExhausted),
      grpcError(Code.ResourceExhausted, ERROR_INFO),
      grpcError(Code.ResourceExhausted, retryInfo(0)),
      new Error('boom')
    );

    expect(waits).toEqual([150, 250, 450, 850]);
  });

  it('encodes a RetryInfo trailer byte for byte as grpc-go does', () => {
    expect(retryInfo(300)).toBe(RETRY_INFO_300MS);
  });

  it.each(scenarios)('shared throttled-call scenario: $name', async (scenario) => {
    const errors = scenario.delays_ms.map((ms) => grpcError(Code.ResourceExhausted, retryInfo(ms)));
    const fn = jest.fn(async () => 'ok');
    for (const error of errors) {
      fn.mockRejectedValueOnce(error);
    }

    const result = retrier(fn, logger);

    if (scenario.succeeds) {
      await expect(result).resolves.toBe('ok');
    } else {
      await expect(result).rejects.toBe(errors[scenario.waits_ms.length]);
    }
    expect(sleepMock.mock.calls.map(([ms]) => ms)).toEqual(scenario.waits_ms);
    expect(fn).toHaveBeenCalledTimes(scenario.waits_ms.length + 1);
  });

  it('after a throttle, later attempts that fail another way keep the throttled attempt limit and back off on their own schedule', async () => {
    const fn = jest
      .fn()
      .mockRejectedValueOnce(grpcError(Code.ResourceExhausted, RETRY_INFO_300MS))
      .mockRejectedValue(grpcError(Code.Unavailable));
    const logError = jest.spyOn(logger, 'error');

    await expect(retrier(fn, logger)).rejects.toThrow('throttled');
    expect(fn).toHaveBeenCalledTimes(MAX_THROTTLED_ATTEMPTS);
    // The server's delay is used once and does not carry over to the later waits.
    expect(sleepMock.mock.calls.map(([ms]) => ms)).toEqual([300, 250, 450, 850]);
    // The retried throttle is not logged as an error; the four other failures are.
    expect(logError).toHaveBeenCalledTimes(MAX_THROTTLED_ATTEMPTS - 1);
  });

  it('keeps the configured attempt count for errors without a RetryInfo delay', async () => {
    const fn = jest.fn().mockRejectedValue(grpcError(Code.Unavailable));

    await expect(retrier(fn, logger)).rejects.toThrow('throttled');
    expect(fn).toHaveBeenCalledTimes(DEFAULT_RETRY_COUNT);
  });
});
