import { Separator } from "@/components/ui/separator";
import { useQuery } from "@tanstack/react-query";
import { queries } from "@/lib/api";
import invariant from "tiny-invariant";
import { useAtom } from "jotai";
import { currTenantAtom } from "@/lib/atoms";
import { Icons } from "@/components/ui/icons";
import { relativeDate } from "@/lib/utils";
import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";

export default function Workers() {
  const [tenant] = useAtom(currTenantAtom);
  invariant(tenant);

  const listWorkersQuery = useQuery({
    ...queries.workers.list(tenant.metadata.id),
  });

  if (listWorkersQuery.isLoading || !listWorkersQuery.data?.rows) {
    return (
      <div className="flex flex-row flex-1 w-full h-full">
        <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />
      </div>
    );
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Workers
        </h2>
        <Separator className="my-4" />
        {/* Grid of workers */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {listWorkersQuery.data?.rows.map((worker) => (
            <div
              key={worker.metadata.id}
              className="border overflow-hidden shadow rounded-lg"
            >
              <div className="px-4 py-5 sm:p-6">
                <h3 className="text-lg leading-6 font-medium text-foreground">
                  {worker.name}
                </h3>
                <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
                  Last seen {relativeDate(worker.lastHeartbeatAt)}
                </p>
              </div>
              <div className="px-4 py-4 sm:px-6">
                <div className="text-sm text-background-secondary">
                  <Link to={`/workers/${worker.metadata.id}`}>
                    <Button>View worker</Button>
                  </Link>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
