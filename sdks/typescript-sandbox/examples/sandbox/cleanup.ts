import { hatchet } from '../hatchet-client';

// > List and delete sandboxes
export async function cleanup(conversationId: string) {
  const sandboxes = await hatchet.sandbox.list({ metadata: { conversation: conversationId } });
  for (const info of sandboxes) {
    console.log(info.sandboxId, info.live ? 'running' : 'stopped');
  }

  const deleted = await hatchet.sandbox.delete({ metadata: { conversation: conversationId } });
  return deleted;
}
// !!
