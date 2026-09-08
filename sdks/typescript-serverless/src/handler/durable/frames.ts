/**
 * One text frame on the durable websocket is one protojson ServerlessDurableFrame. The
 * operator sends first, response and error; the endpoint sends request and done.
 */
import { ServerlessDurableFrame } from '../../generated/proto/v1/serverless';

export function encodeFrame(frame: ServerlessDurableFrame): string {
  return JSON.stringify(ServerlessDurableFrame.toJSON(frame));
}

export function decodeFrame(text: string): ServerlessDurableFrame {
  return ServerlessDurableFrame.fromJSON(JSON.parse(text));
}

/** The name of the oneof member a frame carries, for logs and tests. */
export function frameKind(frame: ServerlessDurableFrame): string {
  if (frame.first) return 'first';
  if (frame.done) return 'done';
  if (frame.error) return 'error';

  if (frame.request) {
    const { request } = frame;
    if (request.memo) return 'memo';
    if (request.completeMemo) return 'completeMemo';
    if (request.waitFor) return 'waitFor';
    if (request.triggerRuns) return 'triggerRuns';
    if (request.evictInvocation) return 'evictInvocation';
    if (request.workerStatus) return 'workerStatus';
    if (request.registerWorker) return 'registerWorker';
    return 'request';
  }

  if (frame.response) {
    const { response } = frame;
    if (response.memoAck) return 'memoAck';
    if (response.waitForAck) return 'waitForAck';
    if (response.triggerRunsAck) return 'triggerRunsAck';
    if (response.entryCompleted) return 'entryCompleted';
    if (response.evictionAck) return 'evictionAck';
    if (response.serverEvict) return 'serverEvict';
    if (response.error) return 'responseError';
    if (response.registerWorker) return 'registerWorkerAck';
    return 'response';
  }

  return 'unknown';
}
