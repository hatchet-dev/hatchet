/* eslint-disable react/no-unstable-nested-components */

import { cn } from '@/next/lib/utils';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/next/components/ui/table';
import {
  Headline,
  HeadlineActionItem,
  HeadlineActions,
  PageTitle,
} from '@/next/components/ui/page-header';
import { Separator } from '@/next/components/ui/separator';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/lib/docs';
import {
  Pagination,
  PageSelector,
  PageSizeSelector,
} from '@/next/components/ui/pagination';
import { Time } from '@/next/components/ui/time';
import { RateLimitsProvider, useRateLimits } from '@/next/hooks/use-ratelimits';

type RateLimitStatus = 'exceeded' | 'warning' | 'active';

type RateLimitStatusConfig = {
  colors: string;
  label: string;
};

const rateLimitStatusConfigs: Record<RateLimitStatus, RateLimitStatusConfig> = {
  exceeded: {
    colors: 'text-red-800 dark:text-red-300 bg-red-500/20 ring-red-500',
    label: 'Exceeded',
  },
  warning: {
    colors:
      'text-orange-800 dark:text-orange-300 bg-orange-500/20 ring-orange-500/30',
    label: 'Warning',
  },
  active: {
    colors:
      'text-green-800 dark:text-green-300 bg-green-500/20 ring-green-500/30',
    label: 'Active',
  },
};

const getStatusBadge = (value: number, limitValue: number) => {
  let status: RateLimitStatus;

  if (value === 0) {
    status = 'exceeded';
  } else if (value <= limitValue * 0.2) {
    // Warning when less than 20% of limit remains
    status = 'warning';
  } else {
    status = 'active';
  }

  const config = rateLimitStatusConfigs[status];

  return (
    <div
      className={cn(
        'inline-flex items-center px-3 py-1 text-xs font-medium rounded-md border-transparent',
        config.colors,
      )}
    >
      {config.label}
    </div>
  );
};

const formatWindow = (window: string) => {
  // Convert window string to human readable format
  const [value, unit] = window.split(' ');
  return `${value} ${unit.toLowerCase()}`;
};

function RateLimitsTable() {
  const { data, isLoading } = useRateLimits();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <p className="text-muted-foreground">Loading rate limits...</p>
      </div>
    );
  }

  if (!data || data.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center p-8 gap-4">
        <p className="text-muted-foreground">No rate limits found.</p>
        <DocsButton doc={docs.home.rate_limits} size="lg" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Status</TableHead>
              <TableHead>Key</TableHead>
              <TableHead>Limit</TableHead>
              <TableHead>Current Value</TableHead>
              <TableHead>Window</TableHead>
              <TableHead>Last Refill</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((rateLimit) => (
              <TableRow
                key={rateLimit.key}
                className="hover:bg-transparent cursor-default"
              >
                <TableCell>
                  {getStatusBadge(rateLimit.value, rateLimit.limitValue)}
                </TableCell>
                <TableCell>{rateLimit.key}</TableCell>
                <TableCell>{rateLimit.limitValue}</TableCell>
                <TableCell>{rateLimit.value}</TableCell>
                <TableCell>{formatWindow(rateLimit.window)}</TableCell>
                <TableCell>
                  <Time date={rateLimit.lastRefill} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Pagination className="justify-between flex flex-row">
        <PageSizeSelector />
        <PageSelector variant="dropdown" />
      </Pagination>
    </div>
  );
}

export default function RateLimitsPage() {
  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Control the rate at which your tasks are executed">
          Rate Limits
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.rate_limits} size="icon" />
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>

      <Separator className="my-4" />

      <div className="space-y-4">
        <RateLimitsProvider>
          <RateLimitsTable />
        </RateLimitsProvider>
      </div>
    </BasicLayout>
  );
}
