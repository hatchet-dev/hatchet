"use server";

import { hatchetClient } from "@/hatchet/client";
import { DEMO_TRIGGERS_OFF, demoMessage, demoTriggersEnabled } from "@/hatchet/demo";
import { echo, sleepThenEcho } from "@/hatchet/tasks";

export interface TriggerResult {
  runId: string;
  output?: unknown;
  error?: string;
}

/**
 * Triggers a run and waits up to 30 s for its output. The declaration is passed by name
 * only: the core client resolves it the way `hatchet.run('echo', ...)` would, and the task's
 * own code runs in the endpoint under app/api/hatchet when the operator delivers it. A demo
 * trigger: it creates runs with the app's token for anyone who can submit the form, so it is
 * off on a deployment unless HATCHET_DEMO_TRIGGERS=true (see hatchet/demo.ts).
 */
export async function triggerTask(formData: FormData): Promise<TriggerResult> {
  if (!demoTriggersEnabled()) {
    return { runId: "", error: DEMO_TRIGGERS_OFF };
  }

  const message = demoMessage(formData.get("message"));
  const task = formData.get("task") === "sleep-then-echo" ? sleepThenEcho : echo;

  const ref = await hatchetClient().runNoWait(task.name, { message });

  try {
    return { runId: ref.workflowRunId, output: await ref.result({ timeoutMs: 30_000 }) };
  } catch (err) {
    return { runId: ref.workflowRunId, error: err instanceof Error ? err.message : String(err) };
  }
}
