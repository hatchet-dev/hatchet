// pi's coding tools wired to Hatchet sandboxes: the extension the agent sees, the document that ties a conversation to
// its sandbox, and the resolver that gives each tool call that sandbox as its environment.
import type { Context } from '@earendil-works/chord/context';
import {
  defineDoc,
  defineExtension,
  type EnvTarget,
  type Extension,
  section,
  type ToolRegistration,
} from '@earendil-works/pi-durable';
import type { ExecutionEnv } from '@earendil-works/pi-durable/env';
import {
  createBashTool,
  createEditTool,
  createReadTool,
  createWriteTool,
} from '@earendil-works/pi-durable/tools';
import type { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import { DEFAULT_WORKSPACE, SandboxEnv, type SandboxEnvOptions } from './sandbox-env.js';

/**
 * Which sandbox a conversation runs in, and the snapshot of its workspace. `owner` tells a fork's inherited copy apart
 * from a sandbox the conversation runs in itself.
 */
export type SandboxState = { owner?: number; sandboxId?: string; snapshotId?: string };

const checkpointEvery32 = (_value: unknown, _ops: unknown, info: { deltasSinceBase: number }) =>
  info.deltasSinceBase >= 32;

/**
 * Rewindable with `fork: "asOf"`: a fork at entry E starts with the snapshot that was current at E, so the forked
 * conversation and its filesystem branch together.
 */
export const SandboxDoc = defineDoc<SandboxState>({
  kind: 'hatchet.sandbox',
  version: 1,
  scope: 'conversation',
  history: 'rewindable',
  fork: 'asOf',
  initial: () => ({}),
  checkpointWhen: checkpointEvery32,
});

/** Records the sandbox's latest snapshot after a tool that can change the workspace. */
export function recordingSnapshot<T extends ToolRegistration>(tool: T): T {
  return {
    ...tool,
    execute: async (args, api, context) => {
      try {
        return await tool.execute(args, api, context);
      } finally {
        const snapshotId =
          api.env instanceof SandboxEnv ? await api.env.latestSnapshot() : undefined;
        if (snapshotId) {
          await api.commit(async (tx) => {
            (await tx.doc(SandboxDoc, api.conversationId)).snapshotId = snapshotId;
          }, context);
        }
      }
    },
  } as T;
}

export type SandboxCodingOptions = {
  /** The extension's name in the registry. Default `hatchet-sandbox-coding`. */
  name?: string;
  /** The system prompt section that tells the agent where it is and how to work. Omit to add none. */
  instructions?: string | (() => string);
};

/** pi's read, write, edit and bash tools. The tools that can change the workspace record its snapshot afterwards. */
export function sandboxCoding(options: SandboxCodingOptions = {}): Extension {
  const { instructions } = options;
  return defineExtension({
    name: options.name ?? 'hatchet-sandbox-coding',
    tools: [
      createReadTool(),
      recordingSnapshot(createWriteTool()),
      recordingSnapshot(createEditTool()),
      recordingSnapshot(createBashTool()),
    ],
    sections:
      instructions === undefined
        ? []
        : [
            section(
              'environment',
              typeof instructions === 'string' ? () => instructions : instructions,
              { tag: false }
            ),
          ],
  });
}

/**
 * The harness's `env` option: each tool call of a conversation runs in the sandbox the conversation owns. A
 * conversation without a sandbox of its own gets no environment, so its tools fail cleanly instead of touching the
 * worker's files.
 */
export function sandboxEnvs(
  hatchet: HatchetClient,
  options: SandboxEnvOptions
): (target: EnvTarget, context: Context) => Promise<ExecutionEnv | undefined> {
  return async ({ conversationId, cwd, read }, context) => {
    const sandbox = await read.snapshot(SandboxDoc, conversationId, context);
    if (!sandbox?.sandboxId || sandbox.owner !== conversationId) return undefined;
    return new SandboxEnv(hatchet, sandbox.sandboxId, {
      ...options,
      cwd: cwd ?? options.workspace ?? DEFAULT_WORKSPACE,
    });
  };
}
