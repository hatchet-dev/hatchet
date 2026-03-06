/**
 * Pre-eviction fallback for DurableContext.
 *
 * Supports engines older than MinEngineVersion.DURABLE_EVICTION.
 * Remove this module when support for those engines is dropped.
 */
import { Conditions, Render } from '@hatchet/v1/conditions';
import { conditionsToPb } from '@hatchet/v1/conditions/transformer';
import { Action as ConditionAction } from '@hatchet/protoc/v1/shared/condition';
import { DurableListenerClient } from '@hatchet/clients/listeners/durable-listener/durable-listener-client';

export async function waitForPreEviction(
  durableListener: DurableListenerClient,
  taskRunExternalId: string,
  waitKey: number,
  conditions: Conditions | Conditions[],
  namespace?: string,
  signal?: AbortSignal
): Promise<{ result: Record<string, any>; nextWaitKey: number }> {
  const pbConditions = conditionsToPb(Render(ConditionAction.CREATE, conditions), namespace);
  const key = `waitFor-${waitKey}`;

  await durableListener.registerDurableEvent({
    taskId: taskRunExternalId,
    signalKey: key,
    sleepConditions: pbConditions.sleepConditions,
    userEventConditions: pbConditions.userEventConditions,
  });

  const event = await durableListener.result(
    { taskId: taskRunExternalId, signalKey: key },
    { signal }
  );

  const eventData =
    event.data instanceof Uint8Array ? new TextDecoder().decode(event.data) : event.data;
  const res = JSON.parse(eventData) as Record<string, Record<string, any>>;
  return { result: res.CREATE, nextWaitKey: waitKey + 1 };
}
