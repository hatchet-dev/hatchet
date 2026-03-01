import { hatchet } from '../../hatchet-client';

const APPROVAL_EVENT_KEY = 'approval:decision';

// > Step 01 Define Approval Task
export const approvalTask = hatchet.durableTask({
  name: 'approval-task',
  executionTimeout: '30m',
  fn: async (_, ctx) => {
    const proposedAction = { action: 'send_email', to: 'user@example.com' };
    const approval = ctx.waitFor({ eventKey: APPROVAL_EVENT_KEY });
    if (approval?.approved) {
      return { status: 'approved', action: proposedAction };
    }
    return { status: 'rejected', reason: approval?.reason ?? '' };
  },
});

// > Step 02 Wait For Event
// Pause until the approval event is pushed. Worker slot is freed while waiting.
type Ctx = { waitFor: (o: { eventKey: string }) => { approved?: boolean; reason?: string } };
function waitForApproval(ctx: Ctx, proposedAction: object) {
  const approval = ctx.waitFor({ eventKey: APPROVAL_EVENT_KEY });
  if (approval?.approved) return { status: 'approved', action: proposedAction };
  return { status: 'rejected', reason: approval?.reason ?? '' };
}
