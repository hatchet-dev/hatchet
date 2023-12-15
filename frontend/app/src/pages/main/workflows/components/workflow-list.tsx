import { Workflow } from "@/lib/api";
import { cn, relativeDate } from "@/lib/utils";
import { ChevronRightIcon } from "@heroicons/react/24/outline";
import { Link } from "react-router-dom";
import { WorkflowTags } from "./workflow-tags";

const statuses = {
  offline: "text-gray-500 bg-gray-100/10",
  online: "text-green-400 bg-green-400/10",
  error: "text-rose-400 bg-rose-400/10",
};

interface WorkflowListProps {
  workflows: Workflow[];
}

export default function WorkflowList({ workflows }: WorkflowListProps) {
  return (
    <ul role="list" className="divide-y divide-white/5">
      {workflows.map((workflow) => (
        <li
          key={workflow.metadata.id}
          className="relative flex items-center space-x-4 py-4 px-4 rounded hover:bg-muted"
        >
          <div className="min-w-0 flex-auto">
            <div className="flex items-center gap-x-3">
              <div
                className={cn(statuses["online"], "flex-none rounded-full p-1")}
              >
                <div className="h-2 w-2 rounded-full bg-current" />
              </div>
              <h2 className="min-w-0 text-sm font-semibold leading-6 text-foreground">
                <Link
                  to={`/workflows/${workflow.metadata.id}`}
                  className="flex gap-x-2"
                >
                  <span className="whitespace-nowrap">{workflow.name}</span>
                  <span className="absolute inset-0" />
                </Link>
              </h2>
            </div>
            <div className="mt-3 flex items-center gap-x-2.5 text-xs leading-5 text-muted-foreground">
              {workflow.lastRun?.metadata.createdAt && (
                <p className="whitespace-nowrap">
                  Last run: {relativeDate(workflow.lastRun?.metadata.createdAt)}
                </p>
              )}
            </div>
          </div>
          <WorkflowTags tags={workflow.tags || []} />
          <ChevronRightIcon
            className="h-5 w-5 flex-none text-muted-foreground"
            aria-hidden="true"
          />
        </li>
      ))}
    </ul>
  );
}
