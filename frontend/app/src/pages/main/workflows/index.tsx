import { Separator } from "@/components/ui/separator";
import WorkflowList from "./components/workflow-list";
import { useQuery } from "@tanstack/react-query";
import { queries } from "@/lib/api";
import invariant from "tiny-invariant";
import { useAtom } from "jotai";
import { currTenantAtom } from "@/lib/atoms";
import { Icons } from "@/components/ui/icons";

export default function Workflows() {
  const [tenant] = useAtom(currTenantAtom);
  invariant(tenant);

  const listWorkflowsQuery = useQuery({
    ...queries.workflows.list(tenant.metadata.id),
  });

  if (listWorkflowsQuery.isLoading) {
    return (
      <div className="flex flex-row flex-1 w-full h-full">
        <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />
      </div>
    );
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight tracking-tight text-foreground">
          Workflows
        </h2>
        <Separator className="my-4" />
        <WorkflowList workflows={listWorkflowsQuery.data?.rows || []} />
      </div>
    </div>
  );
}
