import { DeleteTenantModal } from './delete-tenant-modal';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Loading } from '@/components/v1/ui/loading';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import api from '@/lib/api';
import {
  Organization,
  OrganizationTenant,
  TenantStatusType,
} from '@/lib/api/generated/cloud/data-contracts';
import { appRoutes } from '@/router';
import {
  PlusIcon,
  BuildingOffice2Icon,
  ArrowRightIcon,
} from '@heroicons/react/24/outline';
import { EllipsisVerticalIcon, TrashIcon } from '@heroicons/react/24/outline';
import { useQueries, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { useState } from 'react';

interface TenantsTabProps {
  organization: Organization;
  orgId: string;
}

export function TenantsTab({ organization, orgId }: TenantsTabProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [tenantToArchive, setTenantToArchive] =
    useState<OrganizationTenant | null>(null);

  const tenantQueries = useQueries({
    queries: (organization.tenants || [])
      .filter((tenant) => tenant.status !== TenantStatusType.ARCHIVED)
      .map((tenant) => ({
        queryKey: ['tenant:get', tenant.id],
        queryFn: async () => {
          const result = await api.tenantGet(tenant.id);
          return result.data;
        },
        enabled: !!tenant.id,
      })),
  });

  const tenantsLoading = tenantQueries.some((query) => query.isLoading);
  const detailedTenants = tenantQueries
    .filter((query) => query.data)
    .map((query) => query.data);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium">Tenants</h3>
          <p className="text-sm text-muted-foreground">
            Tenants within this organization
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            navigate({
              to: appRoutes.onboardingCreateTenantRoute.to,
              search: { organizationId: organization.metadata.id },
            });
          }}
          leftIcon={<PlusIcon className="size-4" />}
        >
          Add Tenant
        </Button>
      </div>

      {tenantsLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loading />
        </div>
      ) : organization.tenants && organization.tenants.length > 0 ? (
        <div className="space-y-4">
          <div className="hidden md:block">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>ID</TableHead>
                  <TableHead>Slug</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {organization.tenants
                  .filter(
                    (tenant) => tenant.status !== TenantStatusType.ARCHIVED,
                  )
                  .map((orgTenant) => {
                    const detailedTenant = detailedTenants.find(
                      (t) => t?.metadata.id === orgTenant.id,
                    );
                    return (
                      <TableRow key={orgTenant.id}>
                        <TableCell className="font-medium">
                          {detailedTenant?.name || 'Loading...'}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-sm">
                              {orgTenant.id}
                            </span>
                            <CopyToClipboard text={orgTenant.id} />
                          </div>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {detailedTenant?.slug || '-'}
                        </TableCell>
                        <TableCell>
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-8 w-8 p-0"
                              >
                                <EllipsisVerticalIcon className="size-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem
                                onClick={() => {
                                  navigate({
                                    to: appRoutes.tenantRoute.to,
                                    params: { tenant: orgTenant.id },
                                  });
                                }}
                              >
                                <ArrowRightIcon className="mr-2 size-4" />
                                View Tenant
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => setTenantToArchive(orgTenant)}
                              >
                                <TrashIcon className="mr-2 size-4" />
                                Archive Tenant
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    );
                  })}
              </TableBody>
            </Table>
          </div>

          <div className="space-y-4 md:hidden">
            {organization.tenants
              .filter((tenant) => tenant.status !== TenantStatusType.ARCHIVED)
              .map((orgTenant) => {
                const detailedTenant = detailedTenants.find(
                  (t) => t?.metadata.id === orgTenant.id,
                );
                return (
                  <div
                    key={orgTenant.id}
                    className="space-y-3 rounded-lg border p-4"
                  >
                    <div className="flex items-center justify-between">
                      <h4 className="font-medium">
                        {detailedTenant?.name || 'Loading...'}
                      </h4>
                      <Badge>{orgTenant.status}</Badge>
                    </div>
                    <div className="space-y-2 text-sm">
                      <div>
                        <span className="font-medium text-muted-foreground">
                          Tenant ID:
                        </span>
                        <div className="mt-1 flex items-center gap-2">
                          <span className="font-mono text-sm">
                            {orgTenant.id}
                          </span>
                          <CopyToClipboard text={orgTenant.id} />
                        </div>
                      </div>
                      <div>
                        <span className="font-medium text-muted-foreground">
                          Slug:
                        </span>
                        <span className="ml-2">
                          {detailedTenant?.slug || '-'}
                        </span>
                      </div>
                    </div>
                    <div className="flex justify-end">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-8 w-8 p-0"
                          >
                            <EllipsisVerticalIcon className="size-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => {
                              navigate({
                                to: appRoutes.tenantRoute.to,
                                params: { tenant: orgTenant.id },
                              });
                            }}
                          >
                            <ArrowRightIcon className="mr-2 size-4" />
                            View Tenant
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => setTenantToArchive(orgTenant)}
                          >
                            <TrashIcon className="mr-2 size-4" />
                            Archive Tenant
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </div>
                );
              })}
          </div>
        </div>
      ) : (
        <div className="py-8 text-center">
          <BuildingOffice2Icon className="mx-auto mb-4 h-12 w-12 text-muted-foreground" />
          <h3 className="mb-2 text-lg font-medium">No Tenants Yet</h3>
          <p className="mb-4 text-muted-foreground">
            Add your first tenant to get started.
          </p>
          <Button
            onClick={() => {
              navigate({
                to: appRoutes.onboardingCreateTenantRoute.to,
                search: { organizationId: organization.metadata.id },
              });
            }}
            leftIcon={<PlusIcon className="size-4" />}
          >
            Add Tenant
          </Button>
        </div>
      )}

      {(() => {
        const foundTenant = tenantToArchive
          ? detailedTenants.find((t) => t?.metadata.id === tenantToArchive.id)
          : undefined;
        return (
          tenantToArchive &&
          foundTenant && (
            <DeleteTenantModal
              open={!!tenantToArchive}
              onOpenChange={(open) => !open && setTenantToArchive(null)}
              tenant={tenantToArchive}
              tenantName={foundTenant.name}
              organizationName={organization.name}
              onSuccess={() => {
                queryClient.invalidateQueries({
                  queryKey: ['organization:get', orgId],
                });
                queryClient.invalidateQueries({ queryKey: ['tenant:get'] });
              }}
            />
          )
        );
      })()}
    </div>
  );
}
