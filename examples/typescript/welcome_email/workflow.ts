import { Or } from '@hatchet-dev/typescript-sdk/v1/conditions';
import { durationToMs } from '@hatchet-dev/typescript-sdk/v1/client/duration';
import { hatchet } from '../hatchet-client';

export const ONBOARDING_EVENT_KEY = 'user:onboarding-completed';
const TIMEOUT_SECONDS = 5;
const LOOKBACK_WINDOW = '5m' as const;

// > Models
export type SignupInput = {
  email: string;
  user_id: string;
};

export type WelcomeEmailResult = {
  userId: string;
  welcomeSent: boolean;
  followUpSent: boolean;
};

// > Welcome email task
export const welcomeEmail = hatchet.durableTask<SignupInput, WelcomeEmailResult>({
  name: 'welcome-email',
  onEvents: ['user:signup'],
  executionTimeout: '5m',
  fn: async (input, ctx) => {
    // Step 1: Send the welcome email
    console.log(`Sending welcome email to ${input.email}: finish your first onboarding step`);

    // Step 2: Wait for the user to complete onboarding, or time out
    // (use a longer duration for a more realistic workflow)
    const now = await ctx.now();
    const considerEventsSince = new Date(
      now.getTime() - durationToMs(LOOKBACK_WINDOW)
    ).toISOString();

    const waitResult = await ctx.waitFor(
      Or(
        { sleepFor: `${TIMEOUT_SECONDS}s` },
        // Scope the event condition to this user so that another user's
        // onboarding-completed event does not resolve this wait.
        { eventKey: ONBOARDING_EVENT_KEY, scope: input.user_id, considerEventsSince }
      )
    );

    // The or-group result is { CREATE: { <condition_key>: ... } }.
    // Check whether the onboarding event was the one that resolved.
    const create = (waitResult as Record<string, Record<string, unknown>>)['CREATE'] ?? waitResult;
    const resolvedKey = Object.keys(create as Record<string, unknown>)[0] ?? '';
    const onboardingCompleted = resolvedKey === ONBOARDING_EVENT_KEY;

    if (onboardingCompleted) {
      // Step 3a: User completed onboarding -> skip follow-up
      console.log(`User ${input.user_id} completed onboarding, skipping follow-up`);
      return { userId: input.user_id, welcomeSent: true, followUpSent: false };
    }

    // Step 3b: Timeout -> send follow-up email
    console.log(`Sending follow-up email to ${input.email}: need help finishing onboarding?`);
    return { userId: input.user_id, welcomeSent: true, followUpSent: true };
  },
});
