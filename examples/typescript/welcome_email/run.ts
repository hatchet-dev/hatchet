// > Trigger the workflow
import { hatchet } from '../hatchet-client';
import { welcomeEmail, ONBOARDING_EVENT_KEY, SignupInput } from './workflow';

async function main() {
  const input: SignupInput = {
    email: 'alice@example.com',
    user_id: 'user-123',
  };

  // Start the welcome-email workflow
  const ref = await welcomeEmail.runNoWait(input);
  const runId = await ref.getWorkflowRunId();
  console.log(`Started workflow run: ${runId}`);

  // Push onboarding-completed event (scoped to this user)
  console.log('Pushing onboarding-completed event...');
  await hatchet.events.push(ONBOARDING_EVENT_KEY, { status: 'done' }, { scope: input.user_id });

  // Wait for the workflow to complete
  const result = await ref.output;
  console.log('Workflow completed:', result);
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => {
      process.exit(0);
    });
}
