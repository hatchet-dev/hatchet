import { randomUUID } from 'crypto';
import { makeE2EClient, poll } from '../__e2e__/harness';
import { piSessionManager, PiSessionReply, USER_MESSAGE_EVENT } from './manager';
import {
  MAX_TURNS,
  REPORT_FROM_MEMORY_MESSAGE,
  expectedRecallReply,
  inspectEventKeyMessage,
  inspectTurnTimeoutMessage,
} from './messages';

// Pi ships as ESM, which Jest's CommonJS runtime cannot load, so the worker runs as a
// tsx subprocess rather than in this process through the harness startWorker helper.
const TSX_BIN = require.resolve('tsx/cli');
const WORKER_SCRIPT = 'src/v1/examples/pi_agent/worker.ts';
const HEALTH_PORT = 18011;

// Three Pi turns against a live model provider.
const SESSION_TIMEOUT_MS = 300_000;

// Generous next to the few seconds turn 1 takes to start, and small enough to fail inside
// SESSION_TIMEOUT_MS rather than hanging the suite.
const TURN_ONE_START_TIMEOUT_MS = 120_000;

// Enough tail to span a marker split across chunks without growing without bound.
const OUTPUT_TAIL_CHARS = 64_000;

async function spawnPiWorker() {
  const { spawn } = await import('child_process');
  return spawn(process.execPath, [TSX_BIN, '-r', 'tsconfig-paths/register', WORKER_SCRIPT], {
    cwd: process.cwd(),
    env: {
      ...process.env,
      // Scope Pi's file tools to this directory so it reads the same example copy this
      // test imports rather than another copy elsewhere in the repository.
      PI_WORKSPACE_DIR: __dirname,
      HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED: 'true',
      HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT: String(HEALTH_PORT),
    },
    stdio: 'pipe',
  });
}

type PiWorker = Awaited<ReturnType<typeof spawnPiWorker>>;

// Buffer worker output from spawn so a later wait cannot miss a marker that already
// printed. Reading worker output is test-only, and only to observe the turn markers
// turn.ts logs; the application is driven by events and the reply stream.
function bufferWorkerOutput(proc: PiWorker) {
  let buffered = '';
  const checks = new Set<() => void>();

  const onData = (chunk: Buffer) => {
    buffered = (buffered + chunk.toString()).slice(-OUTPUT_TAIL_CHARS);
    checks.forEach((check) => check());
  };
  proc.stdout?.on('data', onData);
  proc.stderr?.on('data', onData);

  return function waitForOutput(marker: string, timeoutMs: number): Promise<void> {
    if (buffered.includes(marker)) {
      return Promise.resolve();
    }

    return new Promise((resolve, reject) => {
      const check = () => {
        if (!buffered.includes(marker)) {
          return;
        }
        stop();
        resolve();
      };

      const timer = setTimeout(() => {
        stop();
        reject(new Error(`timed out after ${timeoutMs}ms waiting for worker output: ${marker}`));
      }, timeoutMs);

      function stop() {
        checks.delete(check);
        clearTimeout(timer);
      }

      checks.add(check);
    });
  };
}

describe('pi-agent-e2e', () => {
  const hatchet = makeE2EClient();

  async function pushUserMessage(sessionId: string, turn: number, message: string) {
    await hatchet.events.push(USER_MESSAGE_EVENT, { message }, { scope: `${sessionId}:${turn}` });
  }

  it(
    'recalls source values and values introduced only on earlier turns',
    async () => {
      const modelId = process.env.PI_MODEL_ID;
      if (!modelId) {
        console.log('skipping pi-agent-e2e: set PI_MODEL_ID to run it against a Pi provider');
        return;
      }

      const workerProc = await spawnPiWorker();
      const waitForWorkerOutput = bufferWorkerOutput(workerProc);

      try {
        await poll(
          async () => {
            try {
              const resp = await fetch(`http://localhost:${HEALTH_PORT}/health`);
              return resp.ok;
            } catch {
              return false;
            }
          },
          {
            timeoutMs: 60_000,
            intervalMs: 1000,
            shouldStop: (healthy) => healthy === true,
            label: 'pi-agent-worker-health',
          }
        );

        const sessionId = randomUUID();
        const firstRemembered = `alpha-${randomUUID()}`;
        const secondRemembered = `bravo-${randomUUID()}`;

        const ref = await piSessionManager.runNoWait({
          sessionId,
          maxTurns: MAX_TURNS,
          modelProvider: process.env.PI_MODEL_PROVIDER ?? 'anthropic',
          modelId,
        });
        const runId = await ref.getWorkflowRunId();

        // The stream has no replay, so subscription setup starts before the message that
        // lets the session begin work. Nothing acknowledges when it is live.
        const replies = hatchet.runs.subscribeToStream(runId);
        const firstReply = replies.next();

        await pushUserMessage(sessionId, 1, inspectTurnTimeoutMessage(firstRemembered));

        // Wait until turn 1 is executing, then push turn 2 before its wait can register,
        // since the manager only registers that wait once turn 1 completes. Gating on the
        // marker keeps queue time out of the event's age, so it only has to survive the rest
        // of turn 1 and lookback has to recover it.
        await waitForWorkerOutput(`PI_TURN_START ${sessionId} turn=1`, TURN_ONE_START_TIMEOUT_MS);
        await pushUserMessage(sessionId, 2, inspectEventKeyMessage(secondRemembered));

        // A replayed manager republishes a turn, so act on each turn index once. Otherwise
        // this caller could push more than one message into a single turn scope.
        const repliedTurns = new Set<number>();
        for (let next = await firstReply; !next.done; next = await replies.next()) {
          const { turnIndex } = JSON.parse(next.value) as PiSessionReply;
          if (repliedTurns.has(turnIndex)) {
            continue;
          }
          repliedTurns.add(turnIndex);

          if (turnIndex === 2) {
            await pushUserMessage(sessionId, 3, REPORT_FROM_MEMORY_MESSAGE);
          }
        }

        const result = await ref.output;

        // The source values are withheld from every message, so the prompts have to send Pi
        // to the files for them. The remembered values exist only in earlier user messages,
        // so this pins the expected source answers and continuity, not a read invocation.
        expect(result.finalAssistantText).toBe(
          expectedRecallReply(firstRemembered, secondRemembered)
        );
        expect(result.sessionId).toBe(sessionId);
        expect(result.turns.map((t) => t.turnIndex)).toEqual([1, 2, 3]);

        // The caller can only sequence messages if every turn publishes its reply. A
        // replayed manager can republish a logical turn, so this pins which logical turns
        // the caller observed rather than asserting a delivery count.
        expect([...repliedTurns].sort((a, b) => a - b)).toEqual([1, 2, 3]);
      } finally {
        workerProc.kill('SIGTERM');
      }
    },
    SESSION_TIMEOUT_MS
  );
});
