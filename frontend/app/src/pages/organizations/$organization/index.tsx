import { BillingTab } from './components/billing-tab';
import { MembersTab } from './components/members-tab';
import { TenantsTab } from './components/tenants-tab';
import { TokensTab } from './components/tokens-tab';
import { Button } from '@/components/v1/ui/button';
import { Input } from '@/components/v1/ui/input';
import { Loading } from '@/components/v1/ui/loading';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { useOrganizations } from '@/hooks/use-organizations';
import { cloudApi } from '@/lib/api/api';
import { ResourceNotFound } from '@/pages/error/components/resource-not-found';
import { appRoutes } from '@/router';
import { PencilIcon, CheckIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from '@tanstack/react-router';
import { isAxiosError } from 'axios';
import { useState } from 'react';

export default function OrganizationPage() {
  const { organization: orgId } = useParams({
    from: appRoutes.organizationsRoute.to,
  });
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { handleUpdateOrganization, updateOrganizationLoading } =
    useOrganizations();
  const [isEditingName, setIsEditingName] = useState(false);
  const [editedName, setEditedName] = useState('');

  const handleStartEdit = () => {
    if (organizationQuery.data?.name) {
      setEditedName(organizationQuery.data.name);
      setIsEditingName(true);
    }
  };

  const handleCancelEdit = () => {
    setIsEditingName(false);
    setEditedName('');
  };

  const handleSaveEdit = () => {
    if (!orgId || !editedName.trim()) {
      return;
    }

    handleUpdateOrganization(orgId, editedName.trim(), () => {
      setIsEditingName(false);
      setEditedName('');
      queryClient.invalidateQueries({ queryKey: ['organization:get', orgId] });
    });
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSaveEdit();
    } else if (e.key === 'Escape') {
      handleCancelEdit();
    }
  };

  const organizationQuery = useQuery({
    queryKey: ['organization:get', orgId],
    queryFn: async () => {
      if (!orgId) {
        throw new Error('Organization ID is required');
      }
      const result = await cloudApi.organizationGet(orgId);
      return result.data;
    },
    enabled: !!orgId,
  });

  if (organizationQuery.isLoading) {
    return <Loading />;
  }

  if (organizationQuery.isError) {
    const status = isAxiosError(organizationQuery.error)
      ? organizationQuery.error.response?.status
      : undefined;

    if (status === 404 || status === 403) {
      return (
        <ResourceNotFound
          resource="Organization"
          description="The organization you're looking for doesn't exist or you don't have access to it."
          primaryAction={{
            label: 'Dashboard',
            navigate: { to: appRoutes.authenticatedRoute.to },
          }}
        />
      );
    }

    throw organizationQuery.error;
  }

  if (!organizationQuery.data) {
    return <Loading />;
  }

  const organization = organizationQuery.data;

  return (
    <div className="max-h-full overflow-y-auto">
      <div className="mx-auto max-w-6xl space-y-6 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2">
              {isEditingName ? (
                <>
                  <Input
                    value={editedName}
                    onChange={(e) => setEditedName(e.target.value)}
                    onKeyDown={handleKeyPress}
                    className="h-10 px-3 text-2xl font-bold"
                    autoFocus
                    disabled={updateOrganizationLoading}
                  />
                  <Button
                    size="sm"
                    onClick={handleSaveEdit}
                    disabled={updateOrganizationLoading || !editedName.trim()}
                  >
                    <CheckIcon className="size-4" />
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={handleCancelEdit}
                    disabled={updateOrganizationLoading}
                  >
                    <XMarkIcon className="size-4" />
                  </Button>
                </>
              ) : (
                <>
                  <h1 className="text-2xl font-bold">{organization.name}</h1>
                </>
              )}
              {!isEditingName && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={handleStartEdit}
                  className="h-6 w-6 p-0"
                  disabled={updateOrganizationLoading}
                  style={{ opacity: updateOrganizationLoading ? 0.3 : 1 }}
                >
                  <PencilIcon className="size-3" />
                </Button>
              )}
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              const previousPath = sessionStorage.getItem(
                'orgSettingsPreviousPath',
              );
              if (previousPath) {
                sessionStorage.removeItem('orgSettingsPreviousPath');
                navigate({ to: previousPath, replace: false });
              } else {
                window.history.back();
              }
            }}
            className="h-8 w-8 p-0"
          >
            <XMarkIcon className="size-4" />
          </Button>
        </div>

        <Tabs defaultValue="tenants">
          <TabsList layout="underlined">
            <TabsTrigger value="tenants" variant="underlined">
              Tenants
            </TabsTrigger>
            <TabsTrigger value="members" variant="underlined">
              Members & Invites
            </TabsTrigger>
            <TabsTrigger value="tokens" variant="underlined">
              Management Tokens
            </TabsTrigger>
            <TabsTrigger value="billing" variant="underlined">
              Billing
            </TabsTrigger>
          </TabsList>

          <TabsContent value="tenants" className="mt-6">
            <TenantsTab organization={organization} orgId={orgId} />
          </TabsContent>

          <TabsContent value="members" className="mt-6">
            <MembersTab
              organization={organization}
              orgId={orgId}
              onRefetch={() => organizationQuery.refetch()}
            />
          </TabsContent>

          <TabsContent value="tokens" className="mt-6">
            <TokensTab organization={organization} orgId={orgId} />
          </TabsContent>

          <TabsContent value="billing" className="mt-6">
            <BillingTab organization={organization} orgId={orgId} />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
