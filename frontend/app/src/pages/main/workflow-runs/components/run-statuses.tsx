import { Badge } from "@/components/ui/badge";
import { JobRunStatus, StepRunStatus, WorkflowRunStatus } from "@/lib/api";
import { capitalize } from "@/lib/utils";

type RunStatus = `${StepRunStatus | WorkflowRunStatus | JobRunStatus}`;

export function RunStatus({
  status,
  reason,
}: {
  status: RunStatus;
  reason?: string;
}) {
  let variant: "inProgress" | "successful" | "failed" = "inProgress";
  let text = "Running";

  switch (status) {
    case "SUCCEEDED":
      variant = "successful";
      text = "Succeeded";
      break;
    case "FAILED":
    case "CANCELLED":
      variant = "failed";
      text = "Cancelled";

      switch (reason) {
        case "TIMED_OUT":
          text = "Timed out";
          break;
        default:
          break;
      }

      break;
    default:
      break;
  }

  return <Badge variant={variant}>{capitalize(text)}</Badge>;
}
