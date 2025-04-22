import api, {
  UpdateTenantRequest,
  TenantAlertingSettings,
  TenantAlertEmailGroupList,
  CreateTenantAlertEmailGroupRequest,
  TenantAlertEmailGroup,
  ListSlackWebhooks,
} from '@/lib/api';
import {
  useMutation,
  UseMutationResult,
  useQuery,
  UseQueryResult,
} from '@tanstack/react-query';
import useTenant from './use-tenant';
import {
  createContext,
  useContext,
  PropsWithChildren,
  createElement,
  useMemo,
} from 'react';

// Main hook return type
interface TenantAlertsState {
  data?: TenantAlertingSettings;
  emailGroups: {
    list: UseQueryResult<TenantAlertEmailGroupList, Error>;
    create: UseMutationResult<
      TenantAlertEmailGroup,
      Error,
      CreateTenantAlertEmailGroupRequest,
      unknown
    >;
    remove: UseMutationResult<void, Error, string, unknown>;
  };
  slackWebhooks: {
    list: UseQueryResult<ListSlackWebhooks, Error>;
    startUrl: string;
    remove: UseMutationResult<void, Error, string, unknown>;
  };
  isLoading: boolean;
  update: UseMutationResult<
    TenantAlertingSettings,
    Error,
    UpdateTenantRequest,
    unknown
  >;
  createEmailGroup: UseMutationResult<
    TenantAlertEmailGroup,
    Error,
    CreateTenantAlertEmailGroupRequest,
    unknown
  >;
  deleteEmailGroup: UseMutationResult<void, Error, string, unknown>;
}

interface TenantAlertsProviderProps extends PropsWithChildren {
  refetchInterval?: number;
}

const TenantAlertsContext = createContext<TenantAlertsState | null>(null);

export function useTenantAlerts() {
  const context = useContext(TenantAlertsContext);
  if (!context) {
    throw new Error(
      'useTenantAlerts must be used within a TenantAlertsProvider',
    );
  }
  return context;
}

function TenantAlertsProviderContent({ children }: TenantAlertsProviderProps) {
  const { tenant } = useTenant();

  const alertingSettingsQuery = useQuery({
    queryKey: ['tenant-alerting-settings:get', tenant],
    queryFn: async () =>
      (await api.tenantAlertingSettingsGet(tenant?.metadata.id || '')).data,
  });

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenant?.metadata.id || '', data);
      return (await api.tenantAlertingSettingsGet(tenant?.metadata.id || ''))
        .data;
    },
    onSuccess: () => {
      alertingSettingsQuery.refetch();
    },
  });

  const listEmailGroupQuery = useQuery({
    queryKey: ['email-group:list', tenant],
    queryFn: async () =>
      (await api.alertEmailGroupList(tenant?.metadata.id || '')).data,
  });

  const createEmailGroupMutation = useMutation({
    mutationKey: ['email-group:create', tenant],
    mutationFn: async (data: CreateTenantAlertEmailGroupRequest) => {
      const res = await api.alertEmailGroupCreate(
        tenant?.metadata.id || '',
        data,
      );
      return res.data;
    },
    onSuccess: () => {
      listEmailGroupQuery.refetch();
    },
  });

  const deleteEmailGroupMutation = useMutation({
    mutationKey: ['alert-email-group:delete', tenant],
    mutationFn: async (emailGroupId: string) => {
      await api.alertEmailGroupDelete(emailGroupId);
    },
    onSuccess: () => {
      listEmailGroupQuery.refetch();
    },
  });

  const listSlackWebhookQuery = useQuery({
    queryKey: ['slack-webhook:list', tenant],
    queryFn: async () =>
      (await api.slackWebhookList(tenant?.metadata.id || '')).data,
  });

  const startSlackWebhookUrl = useMemo(() => {
    return `/api/v1/tenants/${tenant?.metadata.id}/slack/start`;
  }, [tenant]);

  const deleteSlackWebhookMutation = useMutation({
    mutationKey: ['slack-webhook:delete', tenant],
    mutationFn: async (id: string) => {
      await api.slackWebhookDelete(id);
    },
    onSuccess: () => {
      listSlackWebhookQuery.refetch();
    },
  });

  const value = useMemo(
    () => ({
      data: alertingSettingsQuery.data,
      isLoading: alertingSettingsQuery.isLoading,
      emailGroups: {
        list: listEmailGroupQuery,
        create: createEmailGroupMutation,
        remove: deleteEmailGroupMutation,
      },
      slackWebhooks: {
        list: listSlackWebhookQuery,
        startUrl: startSlackWebhookUrl,
        remove: deleteSlackWebhookMutation,
      },
      update: updateMutation,
      createEmailGroup: createEmailGroupMutation,
      deleteEmailGroup: deleteEmailGroupMutation,
    }),
    [
      alertingSettingsQuery.data,
      alertingSettingsQuery.isLoading,
      updateMutation,
      listEmailGroupQuery,
      createEmailGroupMutation,
      deleteEmailGroupMutation,
      listSlackWebhookQuery,
      startSlackWebhookUrl,
      deleteSlackWebhookMutation,
    ],
  );

  return createElement(TenantAlertsContext.Provider, { value }, children);
}

export function TenantAlertsProvider({
  children,
  refetchInterval,
}: TenantAlertsProviderProps) {
  return (
    <TenantAlertsProviderContent refetchInterval={refetchInterval}>
      {children}
    </TenantAlertsProviderContent>
  );
}
