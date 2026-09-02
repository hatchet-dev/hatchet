import { Status, ServerError } from 'nice-grpc';
import {
  isWorkerStreamInactiveError,
  WORKER_STREAM_INACTIVE_MESSAGE,
} from './grpc-error';

describe('isWorkerStreamInactiveError', () => {
  it('returns true for FailedPrecondition with inactive stream message', () => {
    const err = new ServerError(
      Status.FAILED_PRECONDITION,
      `Heartbeat rejected: ${WORKER_STREAM_INACTIVE_MESSAGE}: worker-1`
    );
    expect(isWorkerStreamInactiveError(err)).toBe(true);
  });

  it('returns false for other FailedPrecondition errors', () => {
    const err = new ServerError(Status.FAILED_PRECONDITION, 'some other precondition');
    expect(isWorkerStreamInactiveError(err)).toBe(false);
  });

  it('returns false for UNAVAILABLE errors', () => {
    const err = new ServerError(Status.UNAVAILABLE, WORKER_STREAM_INACTIVE_MESSAGE);
    expect(isWorkerStreamInactiveError(err)).toBe(false);
  });
});
