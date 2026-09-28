/**
 * `GET /api/echo?message=hello` triggers `echo` and returns the run's output, for curl. The
 * page does the same through a server action.
 */
import { hatchetClient } from "@/hatchet/client";
import { echo } from "@/hatchet/tasks";

export async function GET(request: Request): Promise<Response> {
  const message = new URL(request.url).searchParams.get("message") ?? "hello";
  const ref = await hatchetClient().runNoWait(echo.name, { message });
  const output = await ref.result({ timeoutMs: 30_000 });

  return Response.json({ runId: ref.workflowRunId, output });
}
