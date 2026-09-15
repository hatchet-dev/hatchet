// eslint-disable-next-line @typescript-eslint/ban-ts-comment -- Example file with an external dependency not in the SDK
// @ts-nocheck
import type { SessionEntry } from '@earendil-works/pi-coding-agent';
import { JsonObject } from '@hatchet/v1';
import { hatchet } from '../hatchet-client';

const READ_ONLY_TOOLS = ['read', 'grep', 'find', 'ls'];

export type PiTurnInput = {
  userPrompt: string;
  priorEntries: JsonObject[];
  modelProvider: string;
  modelId: string;
  sessionId?: string;
  turnIndex?: number;
};

export type PiTurnOutput = {
  assistantText: string;
  entries: JsonObject[];
  stopReason: string;
};

export const TURN_EXECUTION_TIMEOUT = '5m';

// > Define the Pi turn task
export const piTurn = hatchet.task({
  name: 'pi-turn',
  executionTimeout: TURN_EXECUTION_TIMEOUT,
  fn: async (input: PiTurnInput, ctx): Promise<PiTurnOutput> => {
    // !!
    ctx.logger.info(`PI_TURN_START ${input.sessionId ?? '-'} turn=${input.turnIndex ?? '-'}`);

    // > Load Pi and resolve the model
    const { createAgentSession, ModelRuntime, SessionManager } =
      await import('@earendil-works/pi-coding-agent');

    const modelRuntime = await ModelRuntime.create();
    const model = modelRuntime.getModel(input.modelProvider, input.modelId);
    if (!model) {
      throw new Error(
        `Pi model ${input.modelProvider}/${input.modelId} is not available. ` +
          'Check that the provider credential is configured for the worker.'
      );
    }
    // !!

    // > Restore and create the Pi session
    const cwd = process.env.PI_WORKSPACE_DIR ?? process.cwd();

    const priorEntries = input.priorEntries as unknown as SessionEntry[];
    const sessionManager =
      priorEntries.length > 0
        ? SessionManager.inMemory(cwd, undefined, priorEntries)
        : SessionManager.inMemory(cwd);

    const { session } = await createAgentSession({
      model,
      modelRuntime,
      sessionManager,
      cwd,
      tools: READ_ONLY_TOOLS,
    });
    // !!

    // > Run the turn and return its entries
    try {
      await session.prompt(input.userPrompt);

      const messages = session.messages as Array<{
        role?: string;
        stopReason?: string;
        errorMessage?: string;
      }>;
      const lastAssistant = [...messages].reverse().find((m) => m.role === 'assistant');
      if (!lastAssistant) {
        throw new Error('Pi turn produced no assistant message');
      }
      const stopReason = lastAssistant.stopReason ?? 'stop';
      if (stopReason === 'error' || stopReason === 'aborted') {
        throw new Error(
          `Pi turn did not complete: stopReason=${stopReason}: ${
            lastAssistant.errorMessage ?? 'no detail'
          }`
        );
      }

      const output = {
        assistantText: session.getLastAssistantText() ?? '',
        entries: sessionManager.getEntries() as unknown as JsonObject[],
        stopReason,
      };
      ctx.logger.info(`PI_TURN_END ${input.sessionId ?? '-'} turn=${input.turnIndex ?? '-'}`);
      return output;
    } finally {
      session.dispose();
    }
    // !!
  },
});
