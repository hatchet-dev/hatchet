import { USER_MESSAGE_EVENT } from './manager';
import { TURN_EXECUTION_TIMEOUT } from './turn';

export const MAX_TURNS = 3;

const TURN_SOURCE_FILE = 'turn.ts';
const MANAGER_SOURCE_FILE = 'manager.ts';

// > Ask Pi to inspect the example source
export function inspectTurnTimeoutMessage(valueToRemember: string): string {
  return (
    `Use your read-only file tools to read ${TURN_SOURCE_FILE} in your workspace directory. ` +
    `Also remember the value ${valueToRemember} for later. Reply with the exact string assigned ` +
    'to the constant TURN_EXECUTION_TIMEOUT and nothing else.'
  );
}

export function inspectEventKeyMessage(valueToRemember: string): string {
  return (
    `Use your read-only file tools to read ${MANAGER_SOURCE_FILE} in your workspace directory. ` +
    `Also remember the value ${valueToRemember} for later. Reply with the exact string assigned ` +
    'to the constant USER_MESSAGE_EVENT and nothing else.'
  );
}

// > Ask Pi to answer from session history alone
export const REPORT_FROM_MEMORY_MESSAGE =
  'Answer from our conversation only. Do not read any files. Reply with these four values in ' +
  'order, separated by single spaces, and nothing else. The TURN_EXECUTION_TIMEOUT value, the ' +
  'first value I asked you to remember, the USER_MESSAGE_EVENT value, then the second value I ' +
  'asked you to remember. Do not add quotes or extra words.';

export function expectedRecallReply(firstRemembered: string, secondRemembered: string): string {
  return [TURN_EXECUTION_TIMEOUT, firstRemembered, USER_MESSAGE_EVENT, secondRemembered].join(' ');
}
