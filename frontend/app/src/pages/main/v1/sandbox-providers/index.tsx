import { CreateProviderDialog } from './components/create-provider-dialog';
import { SandboxGate } from './components/gate';
import { ProviderActionsMenu } from './components/provider-actions';
import { ProviderStatusBadge } from './components/provider-status-badge';
import { useSandboxProviders } from './hooks/use-sandbox-providers';
import { KINDS } from './lib/kinds';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import useCanWrite from '@/hooks/use-can-write';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { SandboxProvider } from '@/lib/api/generated/cloud/data-contracts';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';
import { ColumnDef } from '@tanstack/react-table';
import { useMemo, useState } from 'react';

export default function SandboxProvidersPage() {
  return (
    <SandboxGate>
      <SandboxProvidersPageContent />
    </SandboxGate>
  );
}

function SandboxProvidersPageContent() {
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
  } = useSandboxProviders();
  const [isCreateOpen, setIsCreateOpen] = useState(false);

  const columns = useMemo<ColumnDef<SandboxProvider>[]>(
    () => [
      {
        id: 'name',
        header: 'Name',
        cell: ({ row }) => (
          <Link
            to={appRoutes.tenantSandboxProviderRoute.to}
            params={{
              tenant: tenantId,
              sandboxProvider: row.original.metadata.id,
            }}
            className="font-medium hover:underline"
          >
            {row.original.name}
          </Link>
        ),
      },
      {
        id: 'kind',
        header: 'Vendor',
        cell: ({ row }) => (
          <Badge variant="outline">{KINDS[row.original.kind].label}</Badge>
        ),
      },
      {
        id: 'status',
        header: 'Status',
        cell: ({ row }) => <ProviderStatusBadge provider={row.original} />,
      },
      {
        id: 'template',
        header: 'Default template',
        cell: ({ row }) =>
          row.original.defaultTemplate ? (
            <code className="rounded bg-muted px-2 py-1 font-mono text-xs">
              {row.original.defaultTemplate}
            </code>
          ) : (
            <span className="text-xs text-muted-foreground">—</span>
          ),
      },
      {
        id: 'created',
        header: 'Created',
        cell: ({ row }) => (
          <RelativeDate date={row.original.metadata.createdAt} />
        ),
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => (
          <div className="flex justify-end">
            <ProviderActionsMenu provider={row.original} />
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
        rightActions={
          data.length > 0 && canWrite
            ? [
                <Button
                  key="add"
                  variant="cta"
                  onClick={() => setIsCreateOpen(true)}
                >
                  Connect provider
                </Button>,
              ]
            : []
        }
        emptyState={
          <EmptyState
            title="No sandbox providers"
            description="Connect a sandbox vendor with your own API key. Hatchet then serves the sandbox SDK for this tenant on top of it."
            buttons={
              canWrite
                ? [
                    {
                      label: 'Connect provider',
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
      <CreateProviderDialog
        open={isCreateOpen}
        onOpenChange={setIsCreateOpen}
      />
    </div>
  );
}
