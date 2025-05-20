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
import { useCurrentTenantId } from './use-tenant';
import {
  createContext,
  useContext,
  PropsWithChildren,
  createElement,
  useMemo,
} from 'react';
import { useToast } from './utils/use-toast';

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

export function useTenantDetailsAlerts() {
  const context = useContext(TenantAlertsContext);
  if (!context) {
    throw new Error(
      'useTenantDetailsAlerts must be used within a TenantAlertsProvider',
    );
  }
  return context;
}

function TenantAlertsProviderContent({ children }: TenantAlertsProviderProps) {
  const { tenantId } = useCurrentTenantId();
  const { toast } = useToast();

  const alertingSettingsQuery = useQuery({
    queryKey: ['tenant-alerting-settings:get', tenantId],
    queryFn: async () => {
      try {
        return (await api.tenantAlertingSettingsGet(tenantId)).data;
      } catch (error) {
        toast({
          title: 'Error fetching alert settings',

          variant: 'destructive',
          error,
        });
        return undefined;
      }
    },
  });

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      try {
        await api.tenantUpdate(tenantId, data);
        return (await api.tenantAlertingSettingsGet(tenantId)).data;
      } catch (error) {
        toast({
          title: 'Error updating alert settings',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      alertingSettingsQuery.refetch();
    },
  });

  const listEmailGroupQuery = useQuery({
    queryKey: ['email-group:list', tenantId],
    queryFn: async () => {
      try {
        return (await api.alertEmailGroupList(tenantId)).data;
      } catch (error) {
        toast({
          title: 'Error fetching email groups',

          variant: 'destructive',
          error,
        });
        return { rows: [] };
      }
    },
  });

  const createEmailGroupMutation = useMutation({
    mutationKey: ['email-group:create', tenantId],
    mutationFn: async (data: CreateTenantAlertEmailGroupRequest) => {
      try {
        const res = await api.alertEmailGroupCreate(tenantId, data);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error creating email group',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listEmailGroupQuery.refetch();
    },
  });

  const deleteEmailGroupMutation = useMutation({
    mutationKey: ['alert-email-group:delete', tenantId],
    mutationFn: async (emailGroupId: string) => {
      try {
        await api.alertEmailGroupDelete(emailGroupId);
      } catch (error) {
        toast({
          title: 'Error deleting email group',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listEmailGroupQuery.refetch();
    },
  });

  const listSlackWebhookQuery = useQuery({
    queryKey: ['slack-webhook:list', tenantId],
    queryFn: async () => {
      try {
        return (await api.slackWebhookList(tenantId)).data;
      } catch (error) {
        toast({
          title: 'Error fetching Slack webhooks',

          variant: 'destructive',
          error,
        });
        return { rows: [], pagination: { current_page: 0, num_pages: 0 } };
      }
    },
  });

  const startSlackWebhookUrl = useMemo(() => {
    // FIXME: This would be nice to grab from the generated API, but this can only be done as a request.
    return `/api/v1/tenants/${tenantId}/slack/start`;
  }, [tenantId]);

  const deleteSlackWebhookMutation = useMutation({
    mutationKey: ['slack-webhook:delete', tenantId],
    mutationFn: async (id: string) => {
      try {
        await api.slackWebhookDelete(id);
      } catch (error) {
        toast({
          title: 'Error deleting Slack webhook',

          variant: 'destructive',
          error,
        });
        throw error;
      }
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
