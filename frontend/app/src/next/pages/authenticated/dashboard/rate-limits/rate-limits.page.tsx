import { Badge } from '@/next/components/ui/badge';
import { DataTable } from '@/next/components/runs/runs-table/data-table';

import { FilterProvider } from '@/next/hooks/use-filters';
import {
  Headline,
  HeadlineActionItem,
  HeadlineActions,
  PageTitle,
} from '@/next/components/ui/page-header';
import { Separator } from '@/next/components/ui/separator';
import BasicLayout from '@/next/components/layouts/basic.layout';
import useRateLimits from '@/next/hooks/use-ratelimits';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/docs-meta-data';
import { PaginationProvider, usePagination } from '@/next/hooks/use-pagination';
import { ColumnDef } from '@tanstack/react-table';
import {
  Pagination,
  PageSelector,
  PageSizeSelector,
} from '@/next/components/ui/pagination';
import { RateLimit } from '@/next/lib/api';
import { Time } from '@/next/components/ui/time';

const getStatusBadge = (value: number, limitValue: number) => {
  if (value === 0) {
    return (
      <Badge
        variant="destructive"
        className="bg-red-500 text-white border-red-600"
      >
        Exceeded
      </Badge>
    );
  }
  if (value <= limitValue * 0.2) {
    // Warning when less than 20% of limit remains
    return (
      <Badge
        variant="outline"
        className="bg-orange-500 text-white border-orange-600"
      >
        Warning
      </Badge>
    );
  }
  return (
    <Badge
      variant="outline"
      className="bg-green-500 text-white border-green-600"
    >
      Active
    </Badge>
  );
};

const formatWindow = (window: string) => {
  // Convert window string to human readable format
  const [value, unit] = window.split(' ');
  return `${value} ${unit.toLowerCase()}`;
};

export default function RateLimitsPage() {
  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Control the rate at which your tasks are executed">
          Rate Limits
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home['rate-limits']} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>

      <Separator className="my-4" />

      <div className="space-y-4">
        <FilterProvider>
          <PaginationProvider>
            <RateLimitsTable />
          </PaginationProvider>
        </FilterProvider>
      </div>
    </BasicLayout>
  );
}

function RateLimitsTable() {
  const pagination = usePagination();
  const { data, isLoading } = useRateLimits({
    paginationManager: pagination,
  });

  const rateLimits = data || [];

  const columns: ColumnDef<RateLimit>[] = [
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => {
        const rateLimit = row.original;
        return getStatusBadge(rateLimit.value, rateLimit.limitValue);
      },
    },
    {
      accessorKey: 'key',
      header: 'Key',
    },
    {
      accessorKey: 'limitValue',
      header: 'Limit',
    },
    {
      accessorKey: 'value',
      header: 'Current Value',
    },
    {
      accessorKey: 'window',
      header: 'Window',
      cell: ({ row }) => {
        const rateLimit = row.original;
        return formatWindow(rateLimit.window);
      },
    },
    {
      accessorKey: 'lastRefill',
      header: 'Last Refill',
      cell: ({ row }) => {
        const rateLimit = row.original;
        return <Time date={rateLimit.lastRefill} />;
      },
    },
  ];

  return (
    <>
      <DataTable
        columns={columns}
        data={rateLimits}
        isLoading={isLoading}
        emptyState={
          <div className="flex flex-col items-center justify-center p-8 gap-4">
            <p className="text-muted-foreground">No rate limits found.</p>
            <DocsButton doc={docs.home['rate-limits']} size="lg" />
          </div>
        }
      />

      <Pagination className="p-2 justify-between flex flex-row">
        <PageSizeSelector />
        <PageSelector variant="dropdown" />
      </Pagination>
    </>
  );
}
