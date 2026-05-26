import { randomUUID } from 'crypto';
import { makeE2EClient } from '../__e2e__/harness';
import { welcomeEmail, ONBOARDING_EVENT_KEY, SignupInput } from './workflow';

describe('welcome-email-e2e', () => {
  const hatchet = makeE2EClient();

  it('skips follow-up when onboarding completes before timeout', async () => {
    const userId = `test-${randomUUID().slice(0, 8)}`;
    const input: SignupInput = {
      email: 'alice@example.com',
      user_id: userId,
    };

    const ref = await welcomeEmail.runNoWait(input);

    await hatchet.events.push(ONBOARDING_EVENT_KEY, { status: 'done' }, { scope: userId });

    const result = await ref.output;

    expect((result as any).userId).toBe(userId);
    expect((result as any).welcomeSent).toBe(true);
    expect((result as any).followUpSent).toBe(false);
  }, 120_000);

  it('sends follow-up when timeout fires', async () => {
    const userId = `test-${randomUUID().slice(0, 8)}`;
    const input: SignupInput = {
      email: 'bob@example.com',
      user_id: userId,
    };

    const result = await welcomeEmail.run(input);

    expect((result as any).userId).toBe(userId);
    expect((result as any).welcomeSent).toBe(true);
    expect((result as any).followUpSent).toBe(true);
  }, 120_000);
});
