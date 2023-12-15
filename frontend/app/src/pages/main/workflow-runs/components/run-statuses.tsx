import { Badge } from "@/components/ui/badge";
import { JobRunStatus, StepRunStatus, WorkflowRunStatus } from "@/lib/api";
import { capitalize } from "@/lib/utils";

type RunStatus = `${StepRunStatus | WorkflowRunStatus | JobRunStatus}`;

export function RunStatus({ status }: { status: RunStatus }) {
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
      text = "Failed";
      break;
    default:
      break;
  }

  return <Badge variant={variant}>{capitalize(text)}</Badge>;
}

// {numFailed > 0 && (
//     <Badge
//       variant="secondary"
//       className="rounded-sm px-1 font-normal text-red-400 bg-red-500/20 ring-red-500/30"
//     >
//       {numFailed} Failed
//     </Badge>
//   )}
//   {numSucceeded > 0 && (
//     <Badge
//       variant="secondary"
//       className="rounded-sm px-1 font-normal text-green-400 bg-green-500/20 ring-green-500/30"
//     >
//       {numSucceeded} Succeeded
//     </Badge>
//   )}
//   {numRunning > 0 && (
//     <Badge
//       variant="secondary"
//       className="rounded-sm px-1 font-normal text-yellow-400 bg-yellow-500/20 ring-yellow-500/30"
//     >
//       {numRunning} Running
//     </Badge>
//   )}
