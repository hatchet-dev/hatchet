import { BACKGROUND_CONTEXT, withAbortSignal } from '@earendil-works/chord/context';
import type { ExecOpts } from '@hatchet-dev/sandbox';
import type { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import { describe, expect, it } from 'vitest';
import { SandboxEnv } from '../src/sandbox-env.js';

const EDGE = '\u001e';

function envWith(exec: (cmd: string, opts: ExecOpts) => Promise<unknown>) {
  const hatchet = { sandbox: { connect: async () => ({ exec }) } } as unknown as HatchetClient;
  return new SandboxEnv(hatchet, 'sb', { template: 'node:24-alpine' });
}

describe('SandboxEnv.exec', () => {
  it('runs the command in the sandbox and passes its output on exactly', async () => {
    let sent: { cmd: string; opts: ExecOpts } | undefined;
    const env = envWith(async (cmd, opts) => {
      sent = { cmd, opts };
      return { stdout: `${EDGE} out\n${EDGE}`, stderr: `${EDGE}${EDGE}`, exitCode: 3 };
    });
    const output: string[] = [];
    const abort = new AbortController();
    const result = await env.exec(
      'echo out',
      { timeout: 5, onOutput: (chunk) => void output.push(chunk) },
      withAbortSignal(abort.signal, BACKGROUND_CONTEXT)
    );
    expect(result).toEqual({ ok: true, value: { exitCode: 3 } });
    expect(output).toEqual([' out\n']);
    expect(sent?.cmd).toContain("cd '/workspace' && { echo out");
    expect(sent?.opts).toEqual({ timeoutMs: 5000, signal: abort.signal });
  });

  it('reports an abort while the command runs as aborted', async () => {
    const abort = new AbortController();
    const env = envWith(
      (_cmd, opts) =>
        new Promise((_resolve, reject) => {
          const { signal } = opts;
          if (signal?.aborted) reject(signal.reason);
          signal?.addEventListener('abort', () => reject(signal.reason));
        })
    );
    const pending = env.exec('sleep 60', {}, withAbortSignal(abort.signal, BACKGROUND_CONTEXT));
    await new Promise((resolve) => setTimeout(resolve, 0));
    abort.abort();
    const result = await pending;
    expect(result.ok).toBe(false);
    expect(!result.ok && result.error.code).toBe('aborted');
  });

  it('reports a failed sandbox call as unknown', async () => {
    const env = envWith(async () => {
      throw new Error('provider down');
    });
    const result = await env.exec('true', {}, BACKGROUND_CONTEXT);
    expect(!result.ok && [result.error.code, result.error.message]).toEqual([
      'unknown',
      'provider down',
    ]);
  });
});
