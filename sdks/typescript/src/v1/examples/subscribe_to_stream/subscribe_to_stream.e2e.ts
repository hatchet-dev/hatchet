import sleep from '../../../util/sleep';
import { makeE2EClient } from '../__e2e__/harness';
import {
  dagStream,
  INTER_CHUNK_MS,
  longStream,
  PRE_STREAM_SLEEP_MS,
  STREAM_CHUNKS,
} from './workflow';

async function collectStream(runId: string): Promise<{ elapsedSec: number; chunks: string[] }> {
  const chunks: string[] = [];
  const t0 = Date.now();
  const hatchet = makeE2EClient();
  for await (const chunk of hatchet.runs.subscribeToStream(runId)) {
    chunks.push(chunk);
  }
  return { elapsedSec: (Date.now() - t0) / 1000, chunks };
}

describe('subscribe-to-stream-e2e', () => {
  it(
    'DAG with onFailure: subscribe at start receives every chunk and does not hang up early',
    async () => {
      const ref = await dagStream.runNoWait({});
      const runId = await ref.getWorkflowRunId();
      const result = await collectStream(runId);
      await ref.output;

      expect(result.elapsedSec).toBeGreaterThanOrEqual(1.2);
      expect(result.chunks).toEqual(STREAM_CHUNKS);
    },
    60_000
  );

  it(
    'single task: subscribe at start receives every chunk',
    async () => {
      const ref = await longStream.runNoWait({});
      const runId = await ref.getWorkflowRunId();
      const result = await collectStream(runId);
      await ref.output;

      expect(result.elapsedSec).toBeGreaterThanOrEqual(1.2);
      expect(result.chunks).toEqual(STREAM_CHUNKS);
    },
    60_000
  );

  it(
    'subscribe after the task is RUNNING still receives every chunk',
    async () => {
      const ref = await longStream.runNoWait({});
      const runId = await ref.getWorkflowRunId();
      await sleep(800);
      const result = await collectStream(runId);
      await ref.output;

      expect(result.elapsedSec).toBeGreaterThanOrEqual(1.2);
      expect(result.chunks).toEqual(STREAM_CHUNKS);
    },
    60_000
  );

  it(
    'late join mid-stream stays open and receives the tail instead of hanging up empty',
    async () => {
      const ref = await longStream.runNoWait({});
      const runId = await ref.getWorkflowRunId();
      await sleep(PRE_STREAM_SLEEP_MS + INTER_CHUNK_MS * 8);
      const result = await collectStream(runId);
      await ref.output;

      expect(result.elapsedSec).toBeGreaterThanOrEqual(0.8);
      expect(result.chunks.length).toBeGreaterThan(0);
      expect(STREAM_CHUNKS.join('').endsWith(result.chunks.join(''))).toBe(true);
    },
    60_000
  );
});
