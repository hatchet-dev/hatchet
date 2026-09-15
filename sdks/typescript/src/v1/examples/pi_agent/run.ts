import { randomUUID } from 'crypto';
import { hatchet } from '../hatchet-client';
import { piSessionManager, PiSessionReply, USER_MESSAGE_EVENT } from './manager';
import {
  MAX_TURNS,
  REPORT_FROM_MEMORY_MESSAGE,
  inspectEventKeyMessage,
  inspectTurnTimeoutMessage,
} from './messages';

// > Send a user message
async function sendUserMessage(sessionId: string, turn: number, message: string): Promise<void> {
  await hatchet.events.push(USER_MESSAGE_EVENT, { message }, { scope: `${sessionId}:${turn}` });
}
// !!

// > Drive a Pi session
async function main() {
  const modelId = process.env.PI_MODEL_ID;
  if (!modelId) {
    throw new Error('Set PI_MODEL_ID to a model available to the configured provider.');
  }
  const modelProvider = process.env.PI_MODEL_PROVIDER ?? 'anthropic';

  const sessionId = randomUUID();

  const ref = await piSessionManager.runNoWait({
    sessionId,
    maxTurns: MAX_TURNS,
    modelProvider,
    modelId,
  });
  const runId = await ref.getWorkflowRunId();
  console.log(`Started Pi session run: ${runId}`);

  const replies = hatchet.runs.subscribeToStream(runId);
  const firstReply = replies.next();

  await sendUserMessage(sessionId, 1, inspectTurnTimeoutMessage('Wintergreen'));

  const scriptedMessages = new Map<number, string>([
    [2, inspectEventKeyMessage('4471')],
    [3, REPORT_FROM_MEMORY_MESSAGE],
  ]);
  const handledTurns = new Set<number>();

  for (let next = await firstReply; !next.done; next = await replies.next()) {
    const { turnIndex, reply } = JSON.parse(next.value) as PiSessionReply;

    if (handledTurns.has(turnIndex)) {
      continue;
    }
    handledTurns.add(turnIndex);

    console.log(`Turn ${turnIndex} reply: ${reply}`);

    const nextTurn = turnIndex + 1;
    const nextMessage = scriptedMessages.get(nextTurn);
    if (nextMessage) {
      await sendUserMessage(sessionId, nextTurn, nextMessage);
    }
  }

  const result = await ref.output;
  console.log(`Turns completed: ${JSON.stringify(result.turns)}`);
}
// !!

if (require.main === module) {
  main()
    .then(() => process.exit(0))
    .catch((err) => {
      console.error(err);
      process.exit(1);
    });
}
