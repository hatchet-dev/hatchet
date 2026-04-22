import { CreateEmailGroupDialog } from '../alerting/components/create-email-group-dialog';
import { DeleteEmailGroupForm } from '../alerting/components/delete-email-group-form';
import { DeleteSlackForm } from '../alerting/components/delete-slack-form';
import {
  EmailGroupCell,
  EmailGroupStatusCell,
  EmailGroupActions,
} from '../alerting/components/email-groups-columns';
import { SlackActions } from '../alerting/components/slack-webhooks-columns';
import { UpdateTenantAlertingSettings } from '../alerting/components/update-tenant-alerting-settings-form';
import {
  GithubAccountCell,
  GithubLinkCell,
  GithubSettingsCell,
} from '../github/components/github-installations-columns';
import { CreateSNSDialog } from '../ingestors/components/create-sns-dialog';
import { DeleteSNSForm } from '../ingestors/components/delete-sns-form';
import {
  CopyIngestURL,
  SNSActions,
} from '../ingestors/components/sns-integrations-columns';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import api, {
  CreateSNSIntegrationRequest,
  CreateTenantAlertEmailGroupRequest,
  SNSIntegration,
  SlackWebhook,
  TenantAlertEmailGroup,
  UpdateTenantRequest,
  queries,
} from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { GithubAppInstallation } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError, useApiMetaIntegrations } from '@/lib/hooks';
import { Dialog } from '@radix-ui/react-dialog';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import invariant from 'tiny-invariant';

export default function Integrations() {
  const { cloud } = useCloud();
  const integrations = useApiMetaIntegrations();

  const hasEmailIntegration = integrations?.find((i) => i.name === 'email');
  const hasSlackIntegration = integrations?.find((i) => i.name === 'slack');
  const hasGithubIntegration = cloud?.canLinkGithub;

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <Tabs defaultValue="alerting">
          <TabsList layout="underlined" className="mb-6">
            <TabsTrigger value="alerting" variant="underlined">
              Alerting
            </TabsTrigger>
            <TabsTrigger value="ingestors" variant="underlined">
              Ingestors
            </TabsTrigger>
            {hasGithubIntegration && (
              <TabsTrigger value="github" variant="underlined">
                GitHub
              </TabsTrigger>
            )}
          </TabsList>

          <TabsContent value="alerting">
            <AlertingSettings />
            {hasEmailIntegration && <Separator className="my-6" />}
            {hasEmailIntegration && <EmailGroupsList />}
            {hasSlackIntegration && <Separator className="my-6" />}
            {hasSlackIntegration && <SlackWebhooksList />}
          </TabsContent>

          <TabsContent value="ingestors">
            <SNSIntegrationsList />
          </TabsContent>

          {hasGithubIntegration && (
            <TabsContent value="github">
              <GithubInstallationsList />
            </TabsContent>
          )}
        </Tabs>
      </div>
    </div>
  );
}

const AlertingSettings: React.FC = () => {
  const { tenantId } = useCurrentTenantId();
  const alertingSettings = useQuery({
    ...queries.alertingSettings.get(tenantId),
  });

  const [isLoading, setIsLoading] = useState(false);

  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenantId, data);
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
    <UpdateTenantAlertingSettings
      alertingSettings={alertingSettings.data}
      isLoading={isLoading}
      onSubmit={(opts) => {
        updateMutation.mutate(opts);
      }}
      fieldErrors={{}}
    />
  );
};

function EmailGroupsList() {
  const { tenant } = useTenantDetails();
  const { tenantId } = useCurrentTenantId();
  const [showGroupsDialog, setShowGroupsDialog] = useState(false);
  const [deleteEmailGroup, setDeleteEmailGroup] =
    useState<TenantAlertEmailGroup | null>(null);

  const [isAlertMemberEmails, setIsAlertMemberEmails] = useState(
    tenant?.alertMemberEmails || false,
  );

  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenantId, data);
    },
    onError: handleApiError,
  });

  const listEmailGroupQuery = useQuery({
    ...queries.emailGroups.list(tenantId),
  });

  const groups: TenantAlertEmailGroup[] = useMemo(() => {
    const customGroups = listEmailGroupQuery.data?.rows || [];

    return [
      {
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

  const emailGroupColumns = useMemo(
    () => [
      {
        columnLabel: 'Emails',
        cellRenderer: (group: TenantAlertEmailGroup) => (
          <EmailGroupCell group={group} />
        ),
      },
      {
        columnLabel: 'Created',
        cellRenderer: (group: TenantAlertEmailGroup) =>
          group.metadata.id != 'default' ? (
            <RelativeDate date={group.metadata.createdAt} />
          ) : (
            <div />
          ),
      },
      {
        columnLabel: '',
        cellRenderer: (group: TenantAlertEmailGroup) => (
          <EmailGroupStatusCell
            group={group}
            alertTenantEmailsSet={isAlertMemberEmails}
          />
        ),
      },
      {
        columnLabel: '',
        cellRenderer: (group: TenantAlertEmailGroup) => (
          <EmailGroupActions
            group={group}
            alertTenantEmailsSet={isAlertMemberEmails}
            onDeleteClick={(group) => {
              setDeleteEmailGroup(group);
            }}
            onToggleMembersClick={(value) => {
              setIsAlertMemberEmails(value);
              updateMutation.mutate({
                alertMemberEmails: value,
              });
            }}
          />
        ),
      },
    ],
    [isAlertMemberEmails, updateMutation],
  );

  return (
    <div>
      <div className="flex flex-row items-center justify-between">
        <span className="text-sm font-medium text-muted-foreground">
          Email Groups
        </span>
        <Button
          key="create-email-group"
          onClick={() => {
            setShowGroupsDialog(true);
          }}
        >
          Create new group
        </Button>
      </div>
      <Separator className="my-4" />
      {groups.length > 0 ? (
        <SimpleTable columns={emailGroupColumns} data={groups} />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No email groups found. Create a group to receive alerts via email.
        </div>
      )}
      {showGroupsDialog && (
        <CreateEmailGroup
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
  showGroupDialog,
  setShowGroupDialog,
  onSuccess,
}: {
  onSuccess: () => void;
  showGroupDialog: boolean;
  setShowGroupDialog: (show: boolean) => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createMutation = useMutation({
    mutationKey: ['alert-email-group:create', tenantId],
    mutationFn: async (data: CreateTenantAlertEmailGroupRequest) => {
      const res = await api.alertEmailGroupCreate(tenantId, data);
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
        isLoading={createMutation.isPending}
        onSubmit={createMutation.mutate}
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}

function DeleteEmailGroup({
  emailGroup,
  setShowEmailGroupDelete,
  onSuccess,
}: {
  emailGroup: TenantAlertEmailGroup;
  setShowEmailGroupDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['alert-email-group:delete', tenantId, emailGroup],
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
  const { tenantId } = useCurrentTenantId();
  const { isControlPlaneEnabled } = useControlPlane();
  const [deleteSlack, setDeleteSlack] = useState<SlackWebhook | null>(null);

  const listWebhooksQuery = useQuery({
    ...queries.slackWebhooks.list(tenantId),
  });

  const slackColumns = useMemo(
    () => [
      {
        columnLabel: 'Team',
        cellRenderer: (webhook: SlackWebhook) => <div>{webhook.teamName}</div>,
      },
      {
        columnLabel: 'Channel',
        cellRenderer: (webhook: SlackWebhook) => (
          <div>{webhook.channelName}</div>
        ),
      },
      {
        columnLabel: 'Created',
        cellRenderer: (webhook: SlackWebhook) => (
          <RelativeDate date={webhook.metadata.createdAt} />
        ),
      },
      {
        columnLabel: 'Actions',
        cellRenderer: (webhook: SlackWebhook) => (
          <SlackActions
            webhook={webhook}
            onDeleteClick={(webhook) => {
              setDeleteSlack(webhook);
            }}
          />
        ),
      },
    ],
    [],
  );

  return (
    <div>
      <div className="flex flex-row items-center justify-between">
        <span className="text-sm font-medium text-muted-foreground">
          Slack Webhooks
        </span>
        <a
          href={
            isControlPlaneEnabled
              ? '/api/v1/control-plane/tenants/' + tenantId + '/slack/start'
              : '/api/v1/tenants/' + tenantId + '/slack/start'
          }
        >
          <Button key="add-slack-webhook">Add Slack Webhook</Button>
        </a>
      </div>
      <Separator className="my-4" />
      {(listWebhooksQuery.data?.rows || []).length > 0 ? (
        <SimpleTable
          columns={slackColumns}
          data={listWebhooksQuery.data?.rows || []}
        />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No Slack webhooks found. Add a webhook to receive alerts in Slack.
        </div>
      )}
      {deleteSlack && (
        <DeleteSlackWebhook
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
  slackWebhook,
  setShowSlackWebhookDelete,
  onSuccess,
}: {
  slackWebhook: SlackWebhook;
  setShowSlackWebhookDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['slack-webhook:delete', tenantId, slackWebhook],
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

function SNSIntegrationsList() {
  const { tenantId } = useCurrentTenantId();
  const [showSNSDialog, setShowSNSDialog] = useState(false);
  const [deleteSNS, setDeleteSNS] = useState<SNSIntegration | null>(null);

  const listIntegrationsQuery = useQuery({
    ...queries.snsIntegrations.list(tenantId),
  });

  const snsColumns = useMemo(
    () => [
      {
        columnLabel: 'Topic ARN',
        cellRenderer: (integration: SNSIntegration) => (
          <div>{integration.topicArn}</div>
        ),
      },
      {
        columnLabel: 'Ingest URL',
        cellRenderer: (integration: SNSIntegration) => (
          <CopyIngestURL ingestUrl={integration.ingestUrl || ''} />
        ),
      },
      {
        columnLabel: 'Created',
        cellRenderer: (integration: SNSIntegration) => (
          <RelativeDate date={integration.metadata.createdAt} />
        ),
      },
      {
        columnLabel: 'Actions',
        cellRenderer: (integration: SNSIntegration) => (
          <SNSActions
            integration={integration}
            onDeleteClick={(integration) => {
              setDeleteSNS(integration);
            }}
          />
        ),
      },
    ],
    [],
  );

  return (
    <div>
      <div className="flex flex-row items-center justify-between">
        <span className="text-sm font-medium text-muted-foreground">
          SNS Integrations
        </span>
        <Button onClick={() => setShowSNSDialog(true)}>
          Create SNS Endpoint
        </Button>
      </div>
      <Separator className="my-4" />
      {(listIntegrationsQuery.data?.rows || []).length > 0 ? (
        <SimpleTable
          columns={snsColumns}
          data={listIntegrationsQuery.data?.rows || []}
        />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No SNS integrations found. Create an endpoint to receive events from
          AWS SNS.
        </div>
      )}
      {showSNSDialog && (
        <CreateSNSIntegration
          showSNSDialog={showSNSDialog}
          setShowSNSDialog={setShowSNSDialog}
          onSuccess={() => {
            listIntegrationsQuery.refetch();
          }}
        />
      )}
      {deleteSNS && (
        <DeleteSNSIntegration
          snsIntegration={deleteSNS}
          setShowSNSDelete={() => setDeleteSNS(null)}
          onSuccess={() => {
            setDeleteSNS(null);
            listIntegrationsQuery.refetch();
          }}
        />
      )}
    </div>
  );
}

function CreateSNSIntegration({
  showSNSDialog,
  setShowSNSDialog,
  onSuccess,
}: {
  onSuccess: () => void;
  showSNSDialog: boolean;
  setShowSNSDialog: (show: boolean) => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const [generatedIngestUrl, setGeneratedIngestUrl] = useState<
    string | undefined
  >();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createSNSIntegrationMutation = useMutation({
    mutationKey: ['sns:create', tenantId],
    mutationFn: async (data: CreateSNSIntegrationRequest) => {
      const res = await api.snsCreate(tenantId, data);
      return res.data;
    },
    onSuccess: (data) => {
      setGeneratedIngestUrl(data.ingestUrl);
      onSuccess();
    },
    onError: handleApiError,
  });

  return (
    <Dialog open={showSNSDialog} onOpenChange={setShowSNSDialog}>
      <CreateSNSDialog
        isLoading={createSNSIntegrationMutation.isPending}
        onSubmit={createSNSIntegrationMutation.mutate}
        snsIngestionUrl={generatedIngestUrl}
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}

function DeleteSNSIntegration({
  snsIntegration,
  setShowSNSDelete,
  onSuccess,
}: {
  snsIntegration: SNSIntegration;
  setShowSNSDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['sns:delete', tenantId, snsIntegration],
    mutationFn: async () => {
      await api.snsDelete(snsIntegration.metadata.id);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog open={!!snsIntegration} onOpenChange={setShowSNSDelete}>
      <DeleteSNSForm
        snsIntegration={snsIntegration}
        isLoading={deleteMutation.isPending}
        onSubmit={() => deleteMutation.mutate()}
        onCancel={() => setShowSNSDelete(false)}
      />
    </Dialog>
  );
}

function GithubInstallationsList() {
  const { tenantId } = useCurrentTenantId();
  const { tenant } = useTenantDetails();

  const [installationToLink, setInstallationToLink] = useState<
    string | undefined
  >();

  const listInstallationsQuery = useQuery({
    ...queries.github.listInstallations(tenantId),
  });

  const { handleApiError } = useApiError({});

  const linkInstallationToTenantMutation = useMutation({
    mutationKey: [
      'github-app:update:installation',
      tenantId,
      installationToLink,
    ],
    mutationFn: async () => {
      invariant(installationToLink, 'installationToLink should be set');
      const res = await cloudApi.githubAppUpdateInstallation(
        installationToLink,
        {
          tenant: tenantId,
        },
      );
      return res.data;
    },
    onSuccess: () => {
      setInstallationToLink(undefined);
      listInstallationsQuery.refetch();
    },
    onError: handleApiError,
  });

  const githubColumns = useMemo(
    () => [
      {
        columnLabel: 'Account name',
        cellRenderer: (installation: GithubAppInstallation) => (
          <GithubAccountCell installation={installation} />
        ),
      },
      {
        columnLabel: 'Link to tenant?',
        cellRenderer: (installation: GithubAppInstallation) => (
          <GithubLinkCell
            installation={installation}
            onLinkToTenant={(installationId: string) => {
              setInstallationToLink(installationId);
            }}
          />
        ),
      },
      {
        columnLabel: 'GitHub Settings',
        cellRenderer: (installation: GithubAppInstallation) => (
          <GithubSettingsCell installation={installation} />
        ),
      },
    ],
    [],
  );

  const currentPath = window.location.pathname;

  return (
    <div>
      <div className="flex flex-row items-center justify-between">
        <span className="text-sm font-medium text-muted-foreground">
          GitHub Accounts
        </span>
        <a
          href={`/api/v1/cloud/users/github-app/start?redirect_to=${encodeURIComponent(currentPath)}&with_repo_installation=false`}
        >
          <Button>Link new account</Button>
        </a>
      </div>
      <Separator className="my-4" />
      {(listInstallationsQuery.data?.rows || []).length > 0 ? (
        <SimpleTable
          columns={githubColumns}
          data={listInstallationsQuery.data?.rows || []}
        />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No GitHub accounts linked. Link an account to integrate with CI/CD.
        </div>
      )}
      <ConfirmDialog
        title={`Are you sure?`}
        description={`Linking this app to ${tenant?.name} will allow other members of the tenant to view this installation. Users will only be able to deploy to repositories that they have access to.`}
        submitLabel={'Yes, link to tenant'}
        submitVariant={'default'}
        onSubmit={linkInstallationToTenantMutation.mutate}
        onCancel={() => setInstallationToLink(undefined)}
        isLoading={linkInstallationToTenantMutation.isPending}
        isOpen={installationToLink !== undefined}
      />
    </div>
  );
}
