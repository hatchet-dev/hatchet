import { hatchet } from '../../hatchet-client';

// > Step 03 Push Approval Event
// Include the runId so the event matches the specific task waiting for it.
export async function pushApproval(runId: string, approved: boolean, reason = '') {
  await hatchet.event.push('approval:decision', { runId, approved, reason });
}

// Approve: await pushApproval('run-id-from-ui', true);
// Reject:  await pushApproval('run-id-from-ui', false, 'needs review');
