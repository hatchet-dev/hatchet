import { DurableContext } from '@hatchet-dev/typescript-sdk';
import { hatchet } from '../../hatchet-client';

const APPROVAL_EVENT_KEY = 'approval:decision';

// > Step 02 Wait For Event
function waitForApproval(ctx: DurableContext<unknown>) {
  const runId = ctx.workflowRunId();
  return ctx.waitFor({
    eventKey: APPROVAL_EVENT_KEY,
    expression: `input.runId == '${runId}'`,
  });
}
// !!

// > Step 01 Define Approval Task
export const approvalTask = hatchet.durableTask({
  name: 'approval-task',
  executionTimeout: '30m',
  fn: async (_, ctx) => {
    const proposedAction = { action: 'send_email', to: 'user@example.com' };
    const approval = waitForApproval(ctx);
    if (approval?.approved) {
      return { status: 'approved', action: proposedAction };
    }
    return { status: 'rejected', reason: approval?.reason ?? '' };
  },
});
// !!
