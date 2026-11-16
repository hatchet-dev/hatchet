// pi-durable's ExecutionEnv conformance suite against a real Hatchet sandbox. It needs a Hatchet engine and a sandbox
// provider, so it runs only when PI_SANDBOX_TESTS=1 and HATCHET_CLIENT_TOKEN are set.
import '@hatchet-dev/sandbox';
import { BACKGROUND_CONTEXT as context } from '@earendil-works/chord/context';
import { registerEnvConformance } from '@earendil-works/pi-durable/testing';
import type { Sandbox } from '@hatchet-dev/sandbox';
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import { afterAll, describe, expect, it } from 'vitest';
import { SandboxEnv } from '../src/sandbox-env.js';

const enabled = process.env.PI_SANDBOX_TESTS === '1' && !!process.env.HATCHET_CLIENT_TOKEN;
const TEMPLATE = process.env.PI_SANDBOX_TEMPLATE ?? 'node:24-alpine';
const WORKSPACE = '/workspace';

describe.skipIf(!enabled)('SandboxEnv', () => {
  // One sandbox for the whole suite: each case gets a fresh directory in it.
  let hatchet: HatchetClient | undefined;
  let sandbox: Promise<Sandbox> | undefined;
  let cases = 0;

  const shared = () => {
    hatchet ??= HatchetClient.init();
    sandbox ??= hatchet.sandbox.create({ template: TEMPLATE, idleTimeoutMs: 10 * 60_000 });
    return { hatchet, sandbox };
  };

  afterAll(async () => {
    await (await sandbox)?.kill();
  });

  registerEnvConformance({ describe, expect, it }, 'conformance', async (use) => {
    const { hatchet, sandbox } = shared();
    const { sandboxId } = await sandbox;
    const cwd = `${WORKSPACE}/case-${++cases}`;
    const env = new SandboxEnv(hatchet, sandboxId, {
      template: TEMPLATE,
      workspace: WORKSPACE,
      cwd,
    });
    expect((await env.createDir(cwd, { recursive: true }, context)).ok).toBe(true);
    try {
      await use(env);
    } finally {
      await env.remove(cwd, { recursive: true, force: true }, context);
    }
  });
});
