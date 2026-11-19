// The session worker the e2e tests start as a child process, so a test can kill it. The model is pi-ai's faux
// provider, answering from the transcript alone: a retry on a new process continues the same script.
import type { Message } from '@earendil-works/pi-ai';
import { createModels } from '@earendil-works/pi-ai/models';
import {
  fauxAssistantMessage,
  fauxProvider,
  fauxText,
  fauxToolCall,
  type FauxResponseFactory,
} from '@earendil-works/pi-ai/providers/faux';
import { createRegistry, Harness } from '@earendil-works/pi-durable';
import { openNodeSqliteStorage } from '@earendil-works/pi-durable/storage/sqlite/node';
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import * as path from 'node:path';
import { defineSessionTask, sandboxCoding, sandboxEnvs } from '../../src/index.js';

const TASK = process.env.PI_E2E_TASK!;
const DATA_DIR = process.env.PI_E2E_DATA!;
const TEMPLATE = 'fake';

const textOf = (message: Message | undefined) => {
  const content = message?.content;
  if (typeof content === 'string') return content;
  return (content ?? []).map((block) => (block.type === 'text' ? block.text : '')).join('');
};

/**
 * `bash: <command>` calls the bash tool; `write: <path>` + newline + content calls the write tool. After a tool result
 * the model answers `result: <output>`.
 */
const script: FauxResponseFactory = (context) => {
  const last = context.messages.at(-1);
  if (last?.role === 'toolResult') return fauxAssistantMessage(`result: ${textOf(last).trim()}`);
  const text = textOf(last);
  if (text.startsWith('bash: '))
    return fauxAssistantMessage(fauxToolCall('bash', { command: text.slice('bash: '.length) }), {
      stopReason: 'toolUse',
    });
  if (text.startsWith('write: ')) {
    const [first, ...rest] = text.split('\n');
    return fauxAssistantMessage(
      fauxToolCall('write', { path: first!.slice('write: '.length), content: rest.join('\n') }),
      { stopReason: 'toolUse' }
    );
  }
  return fauxAssistantMessage(fauxText('ok'));
};

const faux = fauxProvider();
faux.setResponses(Array.from({ length: 1000 }, () => script));
const models = createModels();
models.setProvider(faux.provider);

const registry = createRegistry();
registry.install(sandboxCoding());

const hatchet = HatchetClient.init();

const session = defineSessionTask(hatchet, {
  name: TASK,
  agent: { model: { provider: 'faux', modelId: 'faux-1' }, cwd: '/workspace' },
  sandbox: { template: TEMPLATE },
  idleGraceMs: 3_000,
  retries: 3,
  debug: !!process.env.PI_DEBUG,
  openHarness: async (sessionId, onReport, context) =>
    Harness.open(
      await openNodeSqliteStorage(path.join(DATA_DIR, `${sessionId}.sqlite`)),
      { models, registry, env: sandboxEnvs(hatchet, { template: TEMPLATE }), onReport },
      context
    ),
});

const worker = await hatchet.worker(`${TASK}-${process.pid}`, { workflows: [session], slots: 10 });
void worker.start();
await worker.waitUntilReady(30_000);
// The test waits for this line before it relies on the worker.
console.log('PI_E2E_READY');
