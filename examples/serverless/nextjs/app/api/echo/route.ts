/**
 * `GET /api/echo?message=hello` triggers `echo` and returns the run's output, for curl. The
 * page does the same through a server action. Both are demo triggers: anyone who can reach
 * them creates runs with the app's token, so they answer 404 on a deployment unless
 * HATCHET_DEMO_TRIGGERS=true (see hatchet/demo.ts).
 */
import { hatchetClient } from "@/hatchet/client";
import { DEMO_TRIGGERS_OFF, demoMessage, demoTriggersEnabled } from "@/hatchet/demo";
import { echo } from "@/hatchet/tasks";

export async function GET(request: Request): Promise<Response> {
  if (!demoTriggersEnabled()) {
    return Response.json({ error: DEMO_TRIGGERS_OFF }, { status: 404 });
  }

  const message = demoMessage(new URL(request.url).searchParams.get("message"));
  const ref = await hatchetClient().runNoWait(echo.name, { message });
  const output = await ref.result({ timeoutMs: 30_000 });

  return Response.json({ runId: ref.workflowRunId, output });
}
