import { Separator } from '@/components/v1/ui/separator';
import { useMemo, useState } from 'react';
import { useApiError, useApiMetaIntegrations } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import api, {
  CreateTenantAlertEmailGroupRequest,
  SlackWebhook,
  TenantAlertEmailGroup,
  UpdateTenantRequest,
  queries,
} from '@/lib/api';
import { useTenant } from '@/lib/atoms';
import { Spinner } from '@/components/v1/ui/loading';
import { UpdateTenantAlertingSettings } from './components/update-tenant-alerting-settings-form';
import invariant from 'tiny-invariant';
import { columns } from './components/slack-webhooks-columns';
import { columns as emailGroupsColumns } from './components/email-groups-columns';

import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { DeleteSlackForm } from './components/delete-slack-form';
import { Button } from '@/components/v1/ui/button';
import { TenantContextType } from '@/lib/outlet';
import { Dialog } from '@radix-ui/react-dialog';
import { useOutletContext } from 'react-router-dom';
import { CreateEmailGroupDialog } from './components/create-email-group-dialog';
import { DeleteEmailGroupForm } from './components/delete-email-group-form';

export default function Alerting() {
  const integrations = useApiMetaIntegrations();

  const hasEmailIntegration = integrations?.find((i) => i.name === 'email');
  const hasSlackIntegration = integrations?.find((i) => i.name === 'slack');

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-semibold leading-tight text-foreground">
          Alerting
        </h2>
        <p className="text-gray-700 dark:text-gray-300 my-4">
          Manage alerts to get notified on workflow failure.
        </p>
        <Separator className="my-4" />
        <AlertingSettings />
        {hasEmailIntegration && <Separator className="my-4" />}
        {hasEmailIntegration && <EmailGroupsList />}
        {hasSlackIntegration && <Separator className="my-4" />}
        {hasSlackIntegration && <SlackWebhooksList />}
      </div>
    </div>
  );
}

const AlertingSettings: React.FC = () => {
  const { tenant } = useTenant();

  invariant(tenant, 'tenant should be defined');

  const alertingSettings = useQuery({
    ...queries.alertingSettings.get(tenant.metadata.id),
  });

  const [isLoading, setIsLoading] = useState(false);

  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenant.metadata.id, data);
    },
    onMutate: () => {
      setIsLoading(true);
    },
    onSuccess: () => {
      setIsLoading(false);
      alertingSettings.refetch();
    },
    onError: handleApiError,
  });

  if (!alertingSettings.data) {
    return <Spinner />;
  }

  return (
    <div>
      <h3 className="text-xl font-semibold leading-tight text-foreground">
        Settings
      </h3>
      <Separator className="my-4" />
      <div className="flex items-center space-x-2">
        <UpdateTenantAlertingSettings
          alertingSettings={alertingSettings.data}
          isLoading={isLoading}
          onSubmit={(opts) => {
            updateMutation.mutate(opts);
          }}
          fieldErrors={{}}
        />
      </div>
    </div>
  );
};

function EmailGroupsList() {
  const { tenant } = useOutletContext<TenantContextType>();
  const [showGroupsDialog, setShowGroupsDialog] = useState(false);
  const [deleteEmailGroup, setDeleteEmailGroup] =
    useState<TenantAlertEmailGroup | null>(null);

  const [isAlertMemberEmails, setIsAlertMemberEmails] = useState(
    tenant.alertMemberEmails || false,
  );

  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenant.metadata.id, data);
    },
    onError: handleApiError,
  });

  const listEmailGroupQuery = useQuery({
    ...queries.emailGroups.list(tenant.metadata.id),
  });

  const cols = emailGroupsColumns({
    onDeleteClick: (row) => {
      setDeleteEmailGroup(row);
    },
    alertTenantEmailsSet: isAlertMemberEmails,
    onToggleMembersClick: (value) => {
      setIsAlertMemberEmails(value);
      updateMutation.mutate({
        alertMemberEmails: value,
      });
    },
  });

  const groups: TenantAlertEmailGroup[] = useMemo(() => {
    const customGroups = listEmailGroupQuery.data?.rows || [];

    return [
      {
        // Special group for all tenant members
        emails: [],
        metadata: {
          id: 'default',
          createdAt: 'default',
          updatedAt: 'default',
        },
      },
      ...customGroups,
    ];
  }, [listEmailGroupQuery.data]);

  return (
    <div>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Email Groups
        </h3>
        <Button
          key="create-slack-webhook"
          onClick={() => {
            setShowGroupsDialog(true);
          }}
        >
          Create new group
        </Button>
      </div>
      <Separator className="my-4" />
      <DataTable
        isLoading={listEmailGroupQuery.isLoading}
        columns={cols}
        data={groups}
        filters={[]}
        getRowId={(row) => row.metadata.id}
      />
      {showGroupsDialog && (
        <CreateEmailGroup
          tenant={tenant.metadata.id}
          onSuccess={() => {
            setShowGroupsDialog(false);
            listEmailGroupQuery.refetch();
          }}
          showGroupDialog={showGroupsDialog}
          setShowGroupDialog={setShowGroupsDialog}
        />
      )}
      {deleteEmailGroup && (
        <DeleteEmailGroup
          tenant={tenant.metadata.id}
          emailGroup={deleteEmailGroup}
          setShowEmailGroupDelete={() => setDeleteEmailGroup(null)}
          onSuccess={() => {
            setDeleteEmailGroup(null);
            listEmailGroupQuery.refetch();
          }}
        />
      )}
    </div>
  );
}

function CreateEmailGroup({
  tenant,
  showGroupDialog,
  setShowGroupDialog,
  onSuccess,
}: {
  tenant: string;
  onSuccess: () => void;
  showGroupDialog: boolean;
  setShowGroupDialog: (show: boolean) => void;
}) {
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenant],
    mutationFn: async (data: CreateTenantAlertEmailGroupRequest) => {
      const res = await api.alertEmailGroupCreate(tenant, data);
      return res.data;
    },
    onSuccess: () => {
      onSuccess();
    },
    onError: handleApiError,
  });

  return (
    <Dialog open={showGroupDialog} onOpenChange={setShowGroupDialog}>
      <CreateEmailGroupDialog
        isLoading={createTokenMutation.isPending}
        onSubmit={createTokenMutation.mutate}
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}

function DeleteEmailGroup({
  tenant,
  emailGroup,
  setShowEmailGroupDelete,
  onSuccess,
}: {
  tenant: string;
  emailGroup: TenantAlertEmailGroup;
  setShowEmailGroupDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['alert-email-group:delete', tenant, emailGroup],
    mutationFn: async () => {
      await api.alertEmailGroupDelete(emailGroup.metadata.id);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog open={!!emailGroup} onOpenChange={setShowEmailGroupDelete}>
      <DeleteEmailGroupForm
        emailGroup={emailGroup}
        isLoading={deleteMutation.isPending}
        onSubmit={() => deleteMutation.mutate()}
        onCancel={() => setShowEmailGroupDelete(false)}
      />
    </Dialog>
  );
}

function SlackWebhooksList() {
  const { tenant } = useOutletContext<TenantContextType>();
  const [deleteSlack, setDeleteSlack] = useState<SlackWebhook | null>(null);

  const listWebhooksQuery = useQuery({
    ...queries.slackWebhooks.list(tenant.metadata.id),
  });

  const cols = columns({
    onDeleteClick: (row) => {
      setDeleteSlack(row);
    },
  });

  return (
    <div>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Slack Webhooks
        </h3>
        <a href={'/api/v1/tenants/' + tenant.metadata.id + '/slack/start'}>
          <Button key="create-slack-webhook">Add Slack Webhook</Button>
        </a>
      </div>
      <Separator className="my-4" />
      <DataTable
        isLoading={listWebhooksQuery.isLoading}
        columns={cols}
        data={listWebhooksQuery.data?.rows || []}
        filters={[]}
        getRowId={(row) => row.metadata.id}
      />
      {deleteSlack && (
        <DeleteSlackWebhook
          tenant={tenant.metadata.id}
          slackWebhook={deleteSlack}
          setShowSlackWebhookDelete={() => setDeleteSlack(null)}
          onSuccess={() => {
            setDeleteSlack(null);
            listWebhooksQuery.refetch();
          }}
        />
      )}
    </div>
  );
}

function DeleteSlackWebhook({
  tenant,
  slackWebhook,
  setShowSlackWebhookDelete,
  onSuccess,
}: {
  tenant: string;
  slackWebhook: SlackWebhook;
  setShowSlackWebhookDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['slack-webhook:delete', tenant, slackWebhook],
    mutationFn: async () => {
      await api.slackWebhookDelete(slackWebhook.metadata.id);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog open={!!slackWebhook} onOpenChange={setShowSlackWebhookDelete}>
      <DeleteSlackForm
        slackWebhook={slackWebhook}
        isLoading={deleteMutation.isPending}
        onSubmit={() => deleteMutation.mutate()}
        onCancel={() => setShowSlackWebhookDelete(false)}
      />
    </Dialog>
  );
}
