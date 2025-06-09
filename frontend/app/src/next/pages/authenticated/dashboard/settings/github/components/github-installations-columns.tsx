import { ColumnDef } from '@tanstack/react-table';
import { Button } from '@/next/components/ui/button';
import { PlusCircle, CheckCircleIcon } from 'lucide-react';
import {
  GithubAppInstallation,
  GithubIntegrationProvider,
  useGithubIntegration,
} from '@/next/hooks/use-github-integration';
import { Time } from '@/next/components/ui/time';
import { GearIcon } from '@radix-ui/react-icons';
import { Link } from 'react-router-dom';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { Skeleton } from '@/next/components/ui/skeleton';

export const columns = (
  onLinkClick: (installationId: string) => void,
): ColumnDef<GithubAppInstallation>[] => [
  {
    accessorKey: 'account_name',
    header: 'Account',
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <img
          src={row.original.account_avatar_url}
          alt={row.original.account_name}
          className="h-6 w-6 rounded-full"
        />
        <span>{row.original.account_name}</span>
      </div>
    ),
  },
  {
    accessorKey: 'repositories_count',
    header: 'Repositories',
    cell: ({ row }) =>
      row.original.type === 'installation' ? (
        <GithubIntegrationProvider
          initialInstallationId={row.original.metadata.id}
        >
          <GithubRepositoriesList />
        </GithubIntegrationProvider>
      ) : (
        <div>-</div>
      ),
  },
  {
    accessorKey: 'created_at',
    header: 'Created',
    cell: ({ row }) => <Time date={row.original.metadata.createdAt} />,
  },
  {
    id: 'actions',
    cell: ({ row }) => (
      <div className="flex items-center justify-end gap-2">
        {row.original.type === 'installation' ? (
          <>
            <Button
              variant="ghost"
              size="sm"
              disabled={row.original.is_linked_to_tenant}
              onClick={() => onLinkClick(row.original.metadata.id)}
            >
              {row.original.is_linked_to_tenant ? (
                <>
                  <CheckCircleIcon className="h-4 w-4" /> Linked
                </>
              ) : (
                <>
                  <PlusCircle className="h-4 w-4" /> Link to tenant
                </>
              )}
            </Button>
            <Link
              to={`${row.original.installation_settings_url}`}
              target="_blank"
            >
              <Button variant="ghost" size="sm">
                <GearIcon className="h-4 w-4" /> Configure
              </Button>
            </Link>
          </>
        ) : (
          <Link
            to={`${row.original.installation_settings_url}`}
            target="_blank"
          >
            <Button variant="ghost" size="sm">
              <PlusCircle className="h-4 w-4" /> Finish setup
            </Button>
          </Link>
        )}
      </div>
    ),
  },
];

function GithubRepositoriesList() {
  const { repos } = useGithubIntegration();

  if (repos.isLoading) {
    return <Skeleton className="h-4 w-24" />;
  }

  if (!repos.data || repos.data.length === 0) {
    return <div>No repositories</div>;
  }

  const visibleRepos = repos.data.slice(0, 4);
  const remainingCount = repos.data.length - 4;

  return (
    <div className="flex flex-col gap-1">
      {visibleRepos.map((repo) => (
        <div key={repo.repo_name} className="text-sm">
          <Link
            to={`https://github.com/${repo.repo_owner}/${repo.repo_name}`}
            target="_blank"
          >
            {repo.repo_owner}/{repo.repo_name}
          </Link>
        </div>
      ))}
      {remainingCount > 0 && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="text-sm text-muted-foreground cursor-help">
                +{remainingCount} more
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <div className="flex flex-col gap-1 max-h-[200px] overflow-y-auto">
                {repos.data.slice(4).map((repo) => (
                  <div key={repo.repo_name}>
                    <Link
                      to={`https://github.com/${repo.repo_owner}/${repo.repo_name}`}
                      target="_blank"
                    >
                      {repo.repo_owner}/{repo.repo_name}
                    </Link>
                  </div>
                ))}
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}
    </div>
  );
}
