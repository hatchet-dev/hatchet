import {
  createContext,
  useContext,
  PropsWithChildren,
  createElement,
} from 'react';
import useTenant from './use-tenant';
import { useState } from 'react';
import {
  GithubAppInstallation,
  GithubBranch,
  GithubRepo,
} from '@/lib/api/generated/cloud/data-contracts';
import { AxiosError } from 'axios';
import {
  useMutation,
  UseMutationResult,
  useQuery,
} from '@tanstack/react-query';
import { useApiError } from '@/lib/hooks';
import { cloudApi } from '@/lib/api/api';
import { useToast } from './utils/use-toast';

export type { GithubAppInstallation, GithubBranch, GithubRepo };

// Main hook return type
interface GithubIntegrationState {
  installations: {
    data?: GithubAppInstallation[];
    isLoading: boolean;
    refetch: () => Promise<unknown>;
  };
  repos: {
    data?: GithubRepo[];
    isLoading: boolean;
    refetch: () => Promise<unknown>;
  };
  branches: {
    data?: GithubBranch[];
    isLoading: boolean;
    refetch: () => Promise<unknown>;
  };
  linkInstallation: UseMutationResult<
    void,
    AxiosError<unknown, any>,
    string,
    unknown
  >;
  startOAuth: UseMutationResult<void, AxiosError<unknown, any>, void, unknown>;
  selectedInstallation: string | undefined;
  setSelectedInstallation: (installationId: string | undefined) => void;
  selectedRepo: GithubRepo | undefined;
  setSelectedRepo: (repo: GithubRepo | undefined) => void;
}

const GithubIntegrationContext = createContext<GithubIntegrationState | null>(
  null,
);

export function useGithubIntegration() {
  const context = useContext(GithubIntegrationContext);
  if (!context) {
    throw new Error(
      'useGithubIntegration must be used within a GithubIntegrationProvider',
    );
  }
  return context;
}

function GithubIntegrationProviderContent({
  children,
  initialInstallationId,
}: PropsWithChildren<{ initialInstallationId?: string }>) {
  const { tenant } = useTenant();
  const { handleApiError } = useApiError({});
  const { toast } = useToast();

  // State for selected installation and repo
  const [selectedInstallation, setSelectedInstallation] = useState<
    string | undefined
  >(initialInstallationId);
  const [selectedRepo, setSelectedRepo] = useState<GithubRepo>();

  // List installations query
  const listInstallationsQuery = useQuery({
    queryKey: ['github-app:list:installations', tenant?.metadata.id],
    queryFn: async () => {
      if (!tenant?.metadata.id) {
        return { rows: [] };
      }
      try {
        const res = await cloudApi.githubAppListInstallations({
          tenant: tenant.metadata.id,
        });
        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching GitHub installations',

          variant: 'destructive',
          error,
        });
        return { rows: [] };
      }
    },
    enabled: !!tenant?.metadata.id,
  });

  // List repos query
  const listReposQuery = useQuery({
    queryKey: [
      'github-app:list:repos',
      tenant?.metadata.id,
      selectedInstallation,
    ],
    queryFn: async () => {
      if (!tenant?.metadata.id || !selectedInstallation) {
        return [];
      }
      try {
        const res = await cloudApi.githubAppListRepos(selectedInstallation, {
          tenant: tenant.metadata.id,
        });
        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching GitHub repositories',

          variant: 'destructive',
          error,
        });
        return [];
      }
    },
    enabled: !!tenant?.metadata.id && !!selectedInstallation,
  });

  // List branches query
  const listBranchesQuery = useQuery({
    queryKey: [
      'github-app:list:branches',
      tenant?.metadata.id,
      selectedInstallation,
      selectedRepo?.repo_owner,
      selectedRepo?.repo_name,
    ],
    queryFn: async () => {
      if (
        !tenant?.metadata.id ||
        !selectedInstallation ||
        !selectedRepo?.repo_owner ||
        !selectedRepo?.repo_name
      ) {
        return [];
      }
      try {
        const res = await cloudApi.githubAppListBranches(
          selectedInstallation,
          selectedRepo.repo_owner,
          selectedRepo.repo_name,
          {
            tenant: tenant.metadata.id,
          },
        );
        return res.data;
      } catch (error) {
        toast({
          title: 'Error fetching GitHub branches',

          variant: 'destructive',
          error,
        });
        return [];
      }
    },
    enabled:
      !!tenant?.metadata.id &&
      !!selectedInstallation &&
      !!selectedRepo?.repo_owner &&
      !!selectedRepo?.repo_name,
  });

  // Link installation mutation
  const linkInstallationMutation = useMutation({
    mutationKey: ['github-app:update:installation', tenant?.metadata.id],
    mutationFn: async (installationId: string) => {
      try {
        await cloudApi.githubAppUpdateInstallation(installationId, {
          tenant: tenant!.metadata.id,
        });
      } catch (error) {
        toast({
          title: 'Error linking GitHub installation',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listInstallationsQuery.refetch();
    },
    onError: handleApiError,
  });

  // Start OAuth mutation
  const startOAuthMutation = useMutation({
    mutationKey: ['github-app:oauth:start'],
    mutationFn: async () => {
      try {
        await cloudApi.userUpdateGithubAppOauthStart();
      } catch (error) {
        toast({
          title: 'Error starting GitHub OAuth',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onError: handleApiError,
  });

  const value = {
    installations: {
      data: listInstallationsQuery.data?.rows,
      isLoading: listInstallationsQuery.isLoading,
      refetch: listInstallationsQuery.refetch,
    },
    repos: {
      data: listReposQuery.data,
      isLoading: listReposQuery.isLoading,
      refetch: listReposQuery.refetch,
    },
    branches: {
      data: listBranchesQuery.data,
      isLoading: listBranchesQuery.isLoading,
      refetch: listBranchesQuery.refetch,
    },
    linkInstallation: linkInstallationMutation,
    startOAuth: startOAuthMutation,
    selectedInstallation,
    setSelectedInstallation,
    selectedRepo,
    setSelectedRepo,
  };

  return createElement(GithubIntegrationContext.Provider, { value }, children);
}

export function GithubIntegrationProvider({
  children,
  initialInstallationId,
}: PropsWithChildren<{ initialInstallationId?: string }>) {
  return (
    <GithubIntegrationProviderContent
      initialInstallationId={initialInstallationId}
    >
      {children}
    </GithubIntegrationProviderContent>
  );
}
