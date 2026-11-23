// The example's agent: a pi-durable coding agent that fixes a failing test in a small repository. Each session's
// history lives in a SQLite file on the worker's disk, so failover works between workers on this machine.
import { mkdir, readdir, readFile } from 'node:fs/promises';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createModels } from '@earendil-works/pi-ai/models';
import { anthropicProvider } from '@earendil-works/pi-ai/providers/anthropic';
import { createRegistry, Harness } from '@earendil-works/pi-durable';
import { openNodeSqliteStorage } from '@earendil-works/pi-durable/storage/sqlite/node';
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import { defineSessionTask, sandboxCoding, sandboxEnvs } from '@hatchet-dev/pi-durable';

export const hatchet = HatchetClient.init();

const HERE = path.dirname(fileURLToPath(import.meta.url));
const DATA_DIR = process.env.PI_DATA_DIR ?? path.join(HERE, '..', '..', '.pi-data');
const REPO_DIR = path.join(HERE, 'repo');
const TEMPLATE = 'node:24-alpine';
const WORKSPACE = '/workspace';

const models = createModels();
models.setProvider(anthropicProvider()); // reads ANTHROPIC_API_KEY

const registry = createRegistry();
registry.install(
  sandboxCoding({
    instructions: [
      `You are a coding agent. The repository is in ${WORKSPACE}, inside a ${TEMPLATE} sandbox.`,
      'The bash tool runs busybox sh: there is no bash and no git. Run the tests with `npm test`.',
      'Make the smallest change that does the job, then answer in two or three sentences.',
    ].join('\n'),
  })
);

/** The repository every new session starts with. */
async function readRepo(dir = ''): Promise<{ path: string; data: string }[]> {
  const files: { path: string; data: string }[] = [];
  for (const entry of await readdir(path.join(REPO_DIR, dir), { withFileTypes: true })) {
    const rel = path.posix.join(dir, entry.name);
    if (entry.isDirectory()) files.push(...(await readRepo(rel)));
    else files.push({ path: rel, data: await readFile(path.join(REPO_DIR, rel), 'utf8') });
  }
  return files;
}

export const codingSession = defineSessionTask(hatchet, {
  name: 'pi-session',
  agent: {
    model: { provider: 'anthropic', modelId: process.env.PI_MODEL ?? 'claude-opus-5-5' },
    thinkingLevel: 'low',
    cwd: WORKSPACE,
  },
  sandbox: {
    template: TEMPLATE,
    workspace: WORKSPACE,
    seed: async (sandbox) => {
      await sandbox.files.writeFiles(await readRepo());
    },
  },
  maxSessionCostUsd: Number(process.env.PI_MAX_SESSION_COST_USD ?? 3),
  debug: !!process.env.PI_DEBUG,
  openHarness: async (sessionId, onReport, context) => {
    await mkdir(DATA_DIR, { recursive: true });
    return Harness.open(
      await openNodeSqliteStorage(path.join(DATA_DIR, `${sessionId}.sqlite`)),
      {
        models,
        registry,
        settings: {
          retry: { maxRetries: 2 },
          progress: { partialIntervalMs: 250, outputIntervalMs: 250 },
        },
        env: sandboxEnvs(hatchet, { template: TEMPLATE, workspace: WORKSPACE }),
        onReport,
      },
      context
    );
  },
});
