import { t } from './components/copy';
import { CreateEndpointDialog } from './components/create-endpoint-dialog';
import { EndpointActionsMenu } from './components/endpoint-actions';
import { EndpointStatusBadge } from './components/endpoint-status-badge';
import { EntitlementGate } from './components/entitlement-gate';
import { PlatformBadge } from './components/platform-name';
import { useEndpoints } from './hooks/use-endpoints';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import { Loading } from '@/components/v1/ui/loading';
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
  const { data, isLoading, error } = useEndpoints();
  const [isCreateOpen, setIsCreateOpen] = useState(false);

  const columns = useMemo(
    () => [
      {
        columnLabel: t('list.column.name'),
        cellRenderer: (endpoint: V1ServerlessEndpoint) => (
          <div className="flex flex-row items-center gap-2">
            <PlatformBadge kind={endpoint.kind} />
            <Link
              to={appRoutes.tenantEndpointRoute.to}
              params={{ tenant: tenantId, endpoint: endpoint.metadata.id }}
              className="font-medium hover:underline"
            >
              {endpoint.name}
            </Link>
          </div>
        ),
      },
      {
        columnLabel: t('list.column.status'),
        cellRenderer: (endpoint: V1ServerlessEndpoint) => (
          <EndpointStatusBadge endpoint={endpoint} />
        ),
      },
      {
        columnLabel: t('list.column.triggerUrl'),
        cellRenderer: (endpoint: V1ServerlessEndpoint) => (
          <ExpandableUrl url={endpoint.triggerUrl} />
        ),
      },
      {
        columnLabel: t('list.column.statusChanged'),
        cellRenderer: (endpoint: V1ServerlessEndpoint) =>
          endpoint.status.changedAt ? (
            <RelativeDate date={endpoint.status.changedAt} />
          ) : (
            <span className="text-xs text-muted-foreground">—</span>
          ),
      },
      {
        columnLabel: ' ',
        cellRenderer: (endpoint: V1ServerlessEndpoint) => (
          <div className="flex justify-end">
            <EndpointActionsMenu endpoint={endpoint} />
          </div>
        ),
      },
    ],
    [tenantId],
  );

  if (isLoading) {
    return <Loading />;
  }

  if (error) {
    throw error;
  }

  return (
    <div className="flex h-full flex-col">
      {data.length > 0 && canWrite && (
        <div className="mb-4 flex w-full shrink-0 flex-row justify-end">
          <Button variant="cta" onClick={() => setIsCreateOpen(true)}>
            {t('list.add')}
          </Button>
        </div>
      )}
      {data.length > 0 ? (
        <SimpleTable
          columns={columns}
          data={data}
          rowKey={(row) => row.metadata.id}
        />
      ) : (
        <div className="flex flex-1 items-center justify-center">
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
        </div>
      )}
      <CreateEndpointDialog
        open={isCreateOpen}
        onOpenChange={setIsCreateOpen}
      />
    </div>
  );
}
