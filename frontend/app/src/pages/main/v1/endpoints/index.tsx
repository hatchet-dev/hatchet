import { t } from './components/copy';
import { CreateEndpointDialog } from './components/create-endpoint-dialog';
import { EndpointActionsMenu } from './components/endpoint-actions';
import { EndpointStatusBadge } from './components/endpoint-status-badge';
import { EntitlementGate } from './components/entitlement-gate';
import { PlatformBadge } from './components/platform-name';
import { useEndpoints } from './hooks/use-endpoints';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import useCanWrite from '@/hooks/use-can-write';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { V1ServerlessEndpoint } from '@/lib/api';
import { cn } from '@/lib/utils';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';
import { ColumnDef } from '@tanstack/react-table';
import { ChevronRight } from 'lucide-react';
import { useMemo, useState } from 'react';

/** A URL clipped to one line: hover shows it whole, the chevron unfolds it in place. */
const ExpandableUrl = ({ url }: { url: string }) => {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="flex max-w-[420px] flex-row items-start gap-1">
      <button
        type="button"
        onClick={() => setExpanded((e) => !e)}
        aria-label={t(
          expanded ? 'list.triggerUrl.collapse' : 'list.triggerUrl.expand',
        )}
        aria-expanded={expanded}
        className="mt-1 shrink-0 text-muted-foreground hover:text-foreground"
      >
        <ChevronRight
          className={cn(
            'size-3.5 transition-transform',
            expanded && 'rotate-90',
          )}
        />
      </button>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <code
              onClick={() => setExpanded((e) => !e)}
              className={cn(
                'min-w-0 cursor-pointer rounded bg-muted px-2 py-1 font-mono text-xs',
                expanded ? 'whitespace-normal break-all' : 'truncate',
              )}
            >
              {url}
            </code>
          </TooltipTrigger>
          {!expanded && (
            <TooltipContent className="max-w-xl break-all font-mono text-xs">
              {url}
            </TooltipContent>
          )}
        </Tooltip>
      </TooltipProvider>
      <CopyToClipboard text={url} />
    </div>
  );
};

export default function EndpointsPage() {
  return (
    <EntitlementGate>
      <EndpointsPageContent />
    </EntitlementGate>
  );
}

function EndpointsPageContent() {
  const { tenantId } = useCurrentTenantId();
  const canWrite = useCanWrite();
  const {
    data,
    pageCount,
    pagination,
    setPagination,
    setPageSize,
    isLoading,
    isPlaceholderData,
    error,
  } = useEndpoints();
  const [isCreateOpen, setIsCreateOpen] = useState(false);

  const columns = useMemo<ColumnDef<V1ServerlessEndpoint>[]>(
    () => [
      {
        id: 'name',
        header: t('list.column.name'),
        cell: ({ row }) => (
          <div className="flex flex-row items-center gap-2">
            <PlatformBadge kind={row.original.kind} />
            <Link
              to={appRoutes.tenantEndpointRoute.to}
              params={{ tenant: tenantId, endpoint: row.original.metadata.id }}
              className="font-medium hover:underline"
            >
              {row.original.name}
            </Link>
          </div>
        ),
      },
      {
        id: 'status',
        header: t('list.column.status'),
        cell: ({ row }) => <EndpointStatusBadge endpoint={row.original} />,
      },
      {
        id: 'triggerUrl',
        header: t('list.column.triggerUrl'),
        cell: ({ row }) => <ExpandableUrl url={row.original.triggerUrl} />,
      },
      {
        id: 'statusChanged',
        header: t('list.column.statusChanged'),
        cell: ({ row }) =>
          row.original.status.changedAt ? (
            <RelativeDate date={row.original.status.changedAt} />
          ) : (
            <span className="text-xs text-muted-foreground">—</span>
          ),
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => (
          <div className="flex justify-end">
            <EndpointActionsMenu endpoint={row.original} />
          </div>
        ),
      },
    ],
    [tenantId],
  );

  if (error) {
    throw error;
  }

  return (
    <div className="flex h-full flex-col">
      <DataTable
        columns={columns}
        data={data}
        isLoading={isLoading}
        isPlaceholderData={isPlaceholderData}
        // the empty state has its own button
        rightActions={
          data.length > 0 && canWrite
            ? [
                <Button
                  key="add"
                  variant="cta"
                  onClick={() => setIsCreateOpen(true)}
                >
                  {t('list.add')}
                </Button>,
              ]
            : []
        }
        emptyState={
          <EmptyState
            title={t('list.empty.title')}
            description={t('list.empty.description')}
            links={[
              {
                href: t('list.docsUrl'),
                label: t('list.empty.docs'),
                external: true,
              },
            ]}
            buttons={
              canWrite
                ? [
                    {
                      label: t('list.add'),
                      onClick: () => setIsCreateOpen(true),
                    },
                  ]
                : undefined
            }
          />
        }
        showSelectedRows={false}
        pagination={pagination}
        setPagination={setPagination}
        onSetPageSize={setPageSize}
        pageCount={pageCount}
      />
      <CreateEndpointDialog
        open={isCreateOpen}
        onOpenChange={setIsCreateOpen}
      />
    </div>
  );
}
