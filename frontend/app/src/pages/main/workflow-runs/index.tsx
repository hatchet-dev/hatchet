import { DataTable } from "../../../components/molecules/data-table/data-table";
import { columns } from "./components/workflow-runs-columns";
import { Separator } from "@/components/ui/separator";
import { useMemo, useState } from "react";
import {
  ColumnFiltersState,
  PaginationState,
  SortingState,
} from "@tanstack/react-table";
import { useQuery } from "@tanstack/react-query";
import invariant from "tiny-invariant";
import { useAtom } from "jotai";
import { currTenantAtom } from "@/lib/atoms";
import { Icons } from "@/components/ui/icons";
import { queries } from "@/lib/api";

export default function WorkflowRuns() {
  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Workflow Runs
        </h2>
        <Separator className="my-4" />
        <WorkflowRunsTable />
      </div>
    </div>
  );
}

function WorkflowRunsTable() {
  const [tenant] = useAtom(currTenantAtom);
  invariant(tenant);

  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 50,
  });
  const [pageSize, setPageSize] = useState<number>(50);

  const offset = useMemo(() => {
    if (!pagination) {
      return;
    }

    return pagination.pageIndex * pagination.pageSize;
  }, [pagination]);

  const listWorkflowRunsQuery = useQuery({
    ...queries.workflowRuns.list(tenant.metadata.id, {
      offset,
      limit: pageSize,
    }),
  });

  if (listWorkflowRunsQuery.isLoading) {
    return (
      <div className="flex flex-row flex-1 w-full h-full">
        <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />
      </div>
    );
  }

  return (
    <DataTable
      columns={columns}
      data={listWorkflowRunsQuery.data?.rows || []}
      filters={[]}
      sorting={sorting}
      setSorting={setSorting}
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
      pagination={pagination}
      setPagination={setPagination}
      onSetPageSize={setPageSize}
      pageCount={listWorkflowRunsQuery.data?.pagination?.num_pages || 0}
    />
  );
}
