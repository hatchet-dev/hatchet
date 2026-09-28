import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { WorkflowRunEvent, WorkflowRunEventType } from '../../generated/proto/dispatcher';
import type { ServerlessDurableFrame } from '../../generated/proto/v1/serverless';
import { createSocketPair } from '../../testing/socket-pair';
import { RunStreamClosedError, RunWatcher } from '../runs';
import { decodeFrame, encodeFrame, frameKind } from './frames';
import {
  MAX_SOCKET_STREAMS,
  SUBSCRIBE_TO_WORKFLOW_EVENTS,
  SUBSCRIBE_TO_WORKFLOW_RUNS,
  SocketStreams,
  StreamCloseCode,
  StreamOpenError,
  type StreamListener,
} from './streams';

const quiet = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };

function harness(maxStreams?: number) {
  const pair = createSocketPair();
  const received: ServerlessDurableFrame[] = [];

  pair.operator.onMessage((text) => received.push(decodeFrame(text)));

  const streams = new SocketStreams({ socket: pair.endpoint, console: quiet, maxStreams });
  // The invocation routes the operator's frames; here the harness does.
  pair.endpoint.onMessage((text) => streams.handleFrame(decodeFrame(text)));
  const fromOperator = (frame: ServerlessDurableFrame) => pair.operator.send(encodeFrame(frame));
  // Frames cross the pair on a microtask; a macrotask turn drains every one of them.
  const settle = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

  return { pair, received, streams, fromOperator, settle };
}

function listener(): StreamListener & { messages: string[]; closes: Array<[number, string]> } {
  const messages: string[] = [];
  const closes: Array<[number, string]> = [];

  return {
    messages,
    closes,
    onMessage: (message) => messages.push(message),
    onClose: (code, message) => closes.push([code, message]),
  };
}

function finished(workflowRunId: string, outputs: Record<string, unknown>): string {
  return JSON.stringify(
    WorkflowRunEvent.toJSON(
      WorkflowRunEvent.create({
        workflowRunId,
        eventType: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FINISHED,
        results: Object.entries(outputs).map(([taskName, output]) => ({
          taskRunExternalId: `${taskName}-id`,
          taskName,
          jobRunId: 'job',
          output: JSON.stringify(output),
        })),
      })
    )
  );
}

describe('SocketStreams', () => {
  it('opens a stream with its own id, routes its messages and closes it', async () => {
    const { received, streams, fromOperator, settle } = harness();
    const runs = listener();

    const stream = streams.openStream(SUBSCRIBE_TO_WORKFLOW_RUNS, '{"workflowRunId":"a"}', runs);
    await settle();

    expect(stream.id).toBe('1');
    expect(received).toEqual([
      {
        streamOpen: {
          id: '1',
          procedure: SUBSCRIBE_TO_WORKFLOW_RUNS,
          request: '{"workflowRunId":"a"}',
        },
      },
    ]);

    fromOperator({ streamMessage: { id: '1', message: '{"workflowRunId":"a"}' } });
    await settle();
    expect(runs.messages).toEqual(['{"workflowRunId":"a"}']);

    stream.send('{"workflowRunId":"b"}');
    stream.close();
    stream.close();
    await settle();

    expect(received.slice(1)).toEqual([
      { streamMessage: { id: '1', message: '{"workflowRunId":"b"}' } },
      { streamClose: { id: '1', code: 0, message: '' } },
    ]);
    expect(streams.size).toBe(0);
    expect(() => stream.send('{}')).toThrow(/stream 1 .* is closed/);

    // A message crossing the close is dropped, not routed.
    fromOperator({ streamMessage: { id: '1', message: '{"late":true}' } });
    await settle();
    expect(runs.messages).toHaveLength(1);
    expect(runs.closes).toEqual([]);
  });

  it('reports the operator closing a stream once and frees its slot', async () => {
    const { received, streams, fromOperator, settle } = harness();
    const runs = listener();
    const stream = streams.openStream(SUBSCRIBE_TO_WORKFLOW_RUNS, '{}', runs);

    fromOperator({
      streamClose: { id: '1', code: StreamCloseCode.UNAVAILABLE, message: 'engine gone' },
    });
    await settle();

    expect(runs.closes).toEqual([[StreamCloseCode.UNAVAILABLE, 'engine gone']]);
    expect(streams.size).toBe(0);

    // Closing it from the task afterwards sends nothing.
    stream.close();
    await settle();
    expect(received.map(frameKind)).toEqual(['streamOpen']);
  });

  it('refuses a procedure the operator does not serve', () => {
    const { streams } = harness();

    expect(() => streams.openStream('/Dispatcher/Listen', '{}', listener())).toThrow(
      StreamOpenError
    );
    expect(() => streams.openStream('/Dispatcher/Listen', '{}', listener())).toThrow(
      /serves only \/Dispatcher\/SubscribeToWorkflowRuns/
    );
    expect(streams.size).toBe(0);
  });

  it('enforces the per-socket cap of 16 streams and hands the slot back on close', async () => {
    const { streams, settle } = harness();
    const open = Array.from({ length: MAX_SOCKET_STREAMS }, () =>
      streams.openStream(SUBSCRIBE_TO_WORKFLOW_EVENTS, '{}', listener())
    );

    expect(streams.size).toBe(16);
    expect(() => streams.openStream(SUBSCRIBE_TO_WORKFLOW_EVENTS, '{}', listener())).toThrow(
      /already holds 16 streams, the operator's limit/
    );

    open[3].close();
    const next = streams.openStream(SUBSCRIBE_TO_WORKFLOW_EVENTS, '{}', listener());
    await settle();

    expect(next.id).toBe('17');
    expect(streams.size).toBe(16);
  });

  it('closeAll tells the operator about every open stream and refuses further opens', async () => {
    const { received, streams, settle } = harness();
    const first = listener();
    const second = listener();

    streams.openStream(SUBSCRIBE_TO_WORKFLOW_RUNS, '{}', first);
    streams.openStream(SUBSCRIBE_TO_WORKFLOW_EVENTS, '{}', second);
    streams.closeAll();
    streams.closeAll();
    await settle();

    expect(received.slice(2)).toEqual([
      { streamClose: { id: '1', code: 0, message: '' } },
      { streamClose: { id: '2', code: 0, message: '' } },
    ]);
    expect(first.closes).toEqual([]);
    expect(second.closes).toEqual([]);
    expect(streams.size).toBe(0);
    expect(() => streams.openStream(SUBSCRIBE_TO_WORKFLOW_RUNS, '{}', listener())).toThrow(
      /the invocation is over/
    );
  });

  it('fail ends every stream for its listener without sending anything', async () => {
    const { received, streams, settle } = harness();
    const first = listener();
    const second = listener();

    streams.openStream(SUBSCRIBE_TO_WORKFLOW_RUNS, '{}', first);
    streams.openStream(SUBSCRIBE_TO_WORKFLOW_EVENTS, '{}', second);
    streams.fail('the operator closed the socket (code 1006)');
    await settle();

    expect(first.closes).toEqual([
      [StreamCloseCode.UNAVAILABLE, 'the operator closed the socket (code 1006)'],
    ]);
    expect(second.closes).toEqual(first.closes);
    expect(received).toHaveLength(2);
    expect(() => streams.openStream(SUBSCRIBE_TO_WORKFLOW_RUNS, '{}', listener())).toThrow(
      /closed the socket/
    );
  });

  it('cannot open a stream once the socket is closed', async () => {
    const { pair, streams, settle } = harness();

    pair.operator.close(4003, 'cancelled');
    await settle();

    expect(() => streams.openStream(SUBSCRIBE_TO_WORKFLOW_RUNS, '{}', listener())).toThrow(
      /socket is closed/
    );
    expect(streams.size).toBe(0);
  });
});

describe('RunWatcher', () => {
  it('awaits every run on one subscription stream and resolves each by run id', async () => {
    const { received, streams, fromOperator, settle } = harness();
    const watcher = new RunWatcher(streams);

    const a = watcher.awaitRun('run-a');
    const b = watcher.awaitRun('run-b');
    const aAgain = watcher.awaitRun('run-a');
    await settle();

    expect(received).toEqual([
      {
        streamOpen: {
          id: '1',
          procedure: SUBSCRIBE_TO_WORKFLOW_RUNS,
          request: '{"workflowRunId":"run-a"}',
        },
      },
      { streamMessage: { id: '1', message: '{"workflowRunId":"run-b"}' } },
    ]);
    expect(streams.size).toBe(1);

    fromOperator({ streamMessage: { id: '1', message: finished('run-b', { echo: { n: 2 } }) } });
    fromOperator({ streamMessage: { id: '1', message: finished('run-a', { echo: { n: 1 } }) } });
    await settle();

    expect((await b).results[0].output).toBe('{"n":2}');
    expect((await a).workflowRunId).toBe('run-a');
    expect(await aAgain).toBe(await a);
    expect(watcher.subscribed).toBe(true);

    // A run awaited after the stream is up is one more subscription, not a new stream.
    const c = watcher.awaitRun('run-c');
    await settle();
    expect(received).toHaveLength(3);
    expect(received[2]).toEqual({
      streamMessage: { id: '1', message: '{"workflowRunId":"run-c"}' },
    });
    fromOperator({ streamMessage: { id: '1', message: finished('run-c', {}) } });
    expect((await c).results).toEqual([]);
  });

  it('ignores events that are not terminal, for unknown runs, or not events at all', async () => {
    const { streams, fromOperator, settle } = harness();
    const watcher = new RunWatcher(streams);
    let settled = false;

    const wait = watcher.awaitRun('run-a').then(() => {
      settled = true;
    });
    await settle();

    fromOperator({ streamMessage: { id: '1', message: 'not json' } });
    fromOperator({ streamMessage: { id: '1', message: finished('run-other', {}) } });
    fromOperator({
      streamMessage: {
        id: '1',
        message: JSON.stringify({ workflowRunId: 'run-a', eventType: 'UNRECOGNIZED' }),
      },
    });
    await settle();
    expect(settled).toBe(false);

    fromOperator({ streamMessage: { id: '1', message: finished('run-a', {}) } });
    await wait;
    expect(settled).toBe(true);
  });

  it('rejects every waiter when the operator ends the stream, then reopens for the next wait', async () => {
    const { received, streams, fromOperator, settle } = harness();
    const watcher = new RunWatcher(streams);

    const a = watcher.awaitRun('run-a');
    const b = watcher.awaitRun('run-b');
    await settle();

    fromOperator({
      streamClose: { id: '1', code: StreamCloseCode.UNAVAILABLE, message: 'the engine hung up' },
    });
    await settle();

    await expect(a).rejects.toBeInstanceOf(RunStreamClosedError);
    await expect(b).rejects.toThrow(
      /run stream closed before run run-b finished \(code 14: the engine hung up\)/
    );
    expect(watcher.subscribed).toBe(false);

    const c = watcher.awaitRun('run-c');
    await settle();
    expect(received.at(-1)).toEqual({
      streamOpen: {
        id: '2',
        procedure: SUBSCRIBE_TO_WORKFLOW_RUNS,
        request: '{"workflowRunId":"run-c"}',
      },
    });
    fromOperator({ streamMessage: { id: '2', message: finished('run-c', {}) } });
    expect((await c).workflowRunId).toBe('run-c');
  });

  it('rejects with the socket failure when the socket goes away mid-wait', async () => {
    const { streams, settle } = harness();
    const watcher = new RunWatcher(streams);
    const a = watcher.awaitRun('run-a');
    await settle();

    streams.fail('the operator closed the socket (code 4003, cancelled)');

    await expect(a).rejects.toThrow(/code 14: the operator closed the socket \(code 4003/);
  });

  it('cannot wait when the stream cannot be opened', async () => {
    const { streams } = harness(0);
    const watcher = new RunWatcher(streams);

    await expect(watcher.awaitRun('run-a')).rejects.toBeInstanceOf(StreamOpenError);
    expect(watcher.subscribed).toBe(false);
  });

  describe('bounded waits', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      expect(vi.getTimerCount()).toBe(0);
      vi.useRealTimers();
    });

    it('honours the signal and the timeout, and leaves no timer behind', async () => {
      const { streams, fromOperator } = harness();
      const watcher = new RunWatcher(streams);
      const controller = new AbortController();

      const aborted = watcher.awaitRun('run-a', { signal: controller.signal });
      const timedOut = watcher.awaitRun('run-b', { timeoutMs: 1_000 });
      const inTime = watcher.awaitRun('run-c', { timeoutMs: 5_000 });
      await vi.advanceTimersByTimeAsync(0);

      controller.abort();
      await expect(aborted).rejects.toMatchObject({ name: 'AbortError' });

      await vi.advanceTimersByTimeAsync(1_000);
      await expect(timedOut).rejects.toThrow(/timed out after 1000 ms waiting for run run-b/);

      fromOperator({ streamMessage: { id: '1', message: finished('run-c', {}) } });
      await vi.advanceTimersByTimeAsync(0);
      expect((await inTime).workflowRunId).toBe('run-c');

      const alreadyAborted = watcher.awaitRun('run-d', { signal: AbortSignal.abort() });
      await expect(alreadyAborted).rejects.toMatchObject({ name: 'AbortError' });
    });
  });
});
