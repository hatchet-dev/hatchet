import { z } from 'zod/v4';
import { hatchet } from '../hatchet-client';
import { piTurn, PiTurnInput } from './turn';

export const USER_MESSAGE_EVENT = 'pi-session:user-message';
const UserMessagePayload = z.object({ message: z.string() });

const MESSAGE_LOOKBACK = '10m' as const;
// Keep the initial scheduling timeout shorter than MESSAGE_LOOKBACK so the first event cannot fall outside the lookback window.
const SESSION_SCHEDULE_TIMEOUT = '5m' as const;

export type PiSessionManagerInput = {
  sessionId: string;
  maxTurns: number;
  modelProvider: string;
  modelId: string;
};

export type PiSessionReply = {
  turnIndex: number;
  reply: string;
};

export type PiSessionManagerOutput = {
  sessionId: string;
  turns: Array<{ turnIndex: number; entryCount: number }>;
  finalAssistantText: string;
};

// > Define the Pi session manager
export const piSessionManager = hatchet.durableTask({
  name: 'pi-session-manager',
  executionTimeout: '30m',
  scheduleTimeout: SESSION_SCHEDULE_TIMEOUT,
  fn: async (input: PiSessionManagerInput, ctx): Promise<PiSessionManagerOutput> => {
    ctx.logger.info(`PI_SESSION_INVOCATION ${input.sessionId} count=${ctx.invocationCount}`);

    let entries: PiTurnInput['priorEntries'] = [];
    const turns: PiSessionManagerOutput['turns'] = [];
    let finalAssistantText = '';
    // !!

    for (let turnIndex = 1; turnIndex <= input.maxTurns; turnIndex += 1) {
      ctx.logger.info(`PI_SESSION_WAITING ${input.sessionId} turn=${turnIndex}`);
      // > Wait for the next user message
      const { message } = await ctx.waitForEvent(
        USER_MESSAGE_EVENT,
        undefined,
        UserMessagePayload,
        `${input.sessionId}:${turnIndex}`,
        MESSAGE_LOOKBACK,
        `await user message turn ${turnIndex}`
      );
      // !!

      // > Spawn a Pi turn
      const turnInput: PiTurnInput = {
        userPrompt: message,
        priorEntries: entries,
        modelProvider: input.modelProvider,
        modelId: input.modelId,
        sessionId: input.sessionId,
        turnIndex,
      };

      const { entries: turnEntries, assistantText } = await ctx.spawnChild(piTurn, turnInput, {
        key: `${input.sessionId}-turn-${turnIndex}`,
        additionalMetadata: { sessionId: input.sessionId, turnIndex: String(turnIndex) },
      });
      // !!

      entries = turnEntries;
      turns.push({ turnIndex, entryCount: turnEntries.length });
      finalAssistantText = assistantText;

      // > Publish the turn reply
      const reply: PiSessionReply = { turnIndex, reply: assistantText };
      await ctx.putStream(JSON.stringify(reply));
      // !!
    }

    return { sessionId: input.sessionId, turns, finalAssistantText };
  },
});
