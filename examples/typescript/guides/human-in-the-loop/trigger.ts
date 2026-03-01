import { hatchet } from '../../hatchet-client';

// > Step 03 Push Approval Event
// Your frontend or API pushes the approval event when the human clicks Approve/Reject.
// Use the same event key the task is waiting for.
export async function pushApproval(approved: boolean, reason = '') {
  await hatchet.event.push('approval:decision', { approved, reason });
}

// Approve: await pushApproval(true);
// Reject:  await pushApproval(false, 'needs review');
