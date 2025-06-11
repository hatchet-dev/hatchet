import { Label } from '@/next/components/ui/label';
import { useGithubIntegration } from '@/next/hooks/use-github-integration';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/next/components/ui/card';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
} from '@/next/components/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/next/components/ui/popover';
import { Button } from '@/next/components/ui/button';
import { GitHubLogoIcon } from '@radix-ui/react-icons';
import { useState, useEffect } from 'react';
import { GitBranchIcon, GitForkIcon } from 'lucide-react';
import { CreateManagedWorkerBuildConfigRequest } from '@/lib/api/generated/cloud/data-contracts';
import { Spinner } from '@/next/components/ui/spinner';
import { Link } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';

export type GithubRepoSelectorValue = Omit<
  CreateManagedWorkerBuildConfigRequest,
  'steps'
>;

interface GithubRepoSelectorProps {
  value: GithubRepoSelectorValue;
  onChange: (value: GithubRepoSelectorValue) => void;
  actions?: React.ReactNode;
  type?: 'create' | 'update';
}

export function GithubRepoSelector({
  value,
  onChange,
  actions,
  type = 'create',
}: GithubRepoSelectorProps) {
  const {
    installations,
    repos,
    branches,
    selectedInstallation,
    setSelectedInstallation,
    selectedRepo,
    setSelectedRepo,
  } = useGithubIntegration();

  const [openInstallation, setOpenInstallation] = useState(false);
  const [openRepo, setOpenRepo] = useState(false);
  const [openBranch, setOpenBranch] = useState(false);
  const { tenantId } = useCurrentTenantId();

  // Set default installation on first mount
  useEffect(() => {
    const installationsData = installations.data;
    if (
      installationsData &&
      installationsData.length > 0 &&
      !value.githubInstallationId &&
      !selectedInstallation
    ) {
      const firstInstallation = installationsData[0];
      setSelectedInstallation(firstInstallation.metadata.id);
      onChange({
        githubInstallationId: firstInstallation.metadata.id,
        githubRepositoryOwner: '',
        githubRepositoryName: '',
        githubRepositoryBranch: '',
      });
    }
  }, [
    installations.data,
    value.githubInstallationId,
    selectedInstallation,
    setSelectedInstallation,
    onChange,
  ]);

  // Update selectedInstallation when value.installationId changes
  useEffect(() => {
    if (
      value.githubInstallationId &&
      value.githubInstallationId !== selectedInstallation
    ) {
      setSelectedInstallation(value.githubInstallationId);
    }
  }, [
    value.githubInstallationId,
    selectedInstallation,
    setSelectedInstallation,
  ]);

  // Update selectedRepo when value.repo changes
  useEffect(() => {
    if (
      value.githubRepositoryOwner &&
      value.githubRepositoryName &&
      (!selectedRepo ||
        value.githubRepositoryOwner !== selectedRepo.repo_owner ||
        value.githubRepositoryName !== selectedRepo.repo_name)
    ) {
      setSelectedRepo({
        repo_owner: value.githubRepositoryOwner,
        repo_name: value.githubRepositoryName,
      });
    }
  }, [
    value.githubRepositoryOwner,
    value.githubRepositoryName,
    selectedRepo,
    setSelectedRepo,
  ]);

  // Set default branch to 'main' if it exists
  useEffect(() => {
    if (
      branches.data &&
      branches.data.length > 0 &&
      !value.githubRepositoryBranch &&
      value.githubRepositoryOwner &&
      value.githubRepositoryName
    ) {
      const mainBranch = branches.data.find(
        (branch) => branch.branch_name === 'main',
      );
      if (mainBranch) {
        onChange({
          ...value,
          githubRepositoryBranch: 'main',
        });
      }
    }
  }, [branches.data, value, onChange]);

  const selectedInstallationName = installations.data?.find(
    (i) => i.metadata.id === value.githubInstallationId,
  )?.account_name;

  return (
    <Card variant={type === 'update' ? 'borderless' : 'default'}>
      <CardHeader>
        <CardTitle>Worker Source</CardTitle>
        <CardDescription>
          Select the GitHub repository you want to use for your worker pool. If
          you don't see the repository you want to use, please update your{' '}
          <Link to={ROUTES.settings.github(tenantId)} className="underline">
            GitHub integration
          </Link>
          .
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="github-account">GitHub Account</Label>
          <Popover open={openInstallation} onOpenChange={setOpenInstallation}>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                role="combobox"
                aria-expanded={openInstallation}
                className="w-full justify-between"
              >
                <div className="flex items-center gap-2">
                  <GitHubLogoIcon className="h-4 w-4" />
                  {selectedInstallationName ? (
                    selectedInstallationName
                  ) : (
                    <span>Select GitHub account</span>
                  )}
                </div>
              </Button>
            </PopoverTrigger>
            <PopoverContent
              className="w-full p-0 min-w-[200px]"
              side="bottom"
              align="start"
            >
              <Command>
                <CommandInput placeholder="Search accounts..." />
                {!installations.data || installations.data.length === 0 ? (
                  <div className="p-4 text-center">
                    <p className="text-sm text-muted-foreground mb-3">
                      No accounts found.
                    </p>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setOpenInstallation(false);
                        window.location.href = ROUTES.settings.github(tenantId);
                      }}
                      className="w-full"
                    >
                      Link other accounts
                    </Button>
                  </div>
                ) : (
                  <>
                    <CommandEmpty>No accounts found.</CommandEmpty>
                    <CommandGroup className="max-h-[300px] overflow-auto">
                      {installations.data?.map((installation) => (
                        <CommandItem
                          key={installation.metadata.id}
                          onSelect={() => {
                            onChange({
                              githubInstallationId: installation.metadata.id,
                              githubRepositoryOwner: '',
                              githubRepositoryName: '',
                              githubRepositoryBranch: '',
                            });
                            setOpenInstallation(false);
                          }}
                        >
                          {installation.account_name}
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  </>
                )}
              </Command>
            </PopoverContent>
          </Popover>
        </div>

        <div className="space-y-2">
          <Label htmlFor="github-repo">Repository</Label>
          <Popover open={openRepo} onOpenChange={setOpenRepo}>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                role="combobox"
                aria-expanded={openRepo}
                className="w-full justify-between"
                disabled={!value.githubInstallationId}
              >
                <div className="flex items-center gap-2">
                  <GitForkIcon className="h-4 w-4" />
                  {value.githubRepositoryOwner && value.githubRepositoryName ? (
                    <span>
                      {value.githubRepositoryOwner}/{value.githubRepositoryName}
                    </span>
                  ) : (
                    <span>Select repository</span>
                  )}
                </div>
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-full p-0">
              <Command>
                <CommandInput placeholder="Search repositories..." />
                <CommandEmpty>No repositories found.</CommandEmpty>
                <CommandGroup className="max-h-[300px] overflow-auto">
                  {repos.data?.map((repo) => (
                    <CommandItem
                      key={`${repo.repo_owner}/${repo.repo_name}`}
                      onSelect={() => {
                        onChange({
                          ...value,
                          githubRepositoryOwner: repo.repo_owner,
                          githubRepositoryName: repo.repo_name,
                          githubRepositoryBranch: '',
                        });
                        setOpenRepo(false);
                      }}
                    >
                      {repo.repo_owner}/{repo.repo_name}
                    </CommandItem>
                  ))}
                </CommandGroup>
              </Command>
            </PopoverContent>
          </Popover>
        </div>

        <div className="space-y-2">
          <Label htmlFor="github-branch">Branch</Label>
          <Popover open={openBranch} onOpenChange={setOpenBranch}>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                role="combobox"
                aria-expanded={openBranch}
                className="w-full justify-between"
                loading={branches.isLoading}
                disabled={
                  !value.githubRepositoryOwner ||
                  !value.githubRepositoryName ||
                  branches.isLoading
                }
              >
                <div className="flex items-center gap-2">
                  <GitBranchIcon className="h-4 w-4" />
                  {branches.isLoading ? (
                    <Spinner className="h-4 w-4" />
                  ) : value.githubRepositoryBranch ? (
                    value.githubRepositoryBranch
                  ) : (
                    <span>Select branch</span>
                  )}
                </div>
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-full p-0">
              <Command>
                <CommandInput placeholder="Search branches..." />
                <CommandEmpty>No branches found.</CommandEmpty>
                <CommandGroup className="max-h-[300px] overflow-auto">
                  {branches.data?.map((branch) => (
                    <CommandItem
                      key={branch.branch_name}
                      onSelect={() => {
                        onChange({
                          ...value,
                          githubRepositoryBranch: branch.branch_name,
                        });
                        setOpenBranch(false);
                      }}
                    >
                      {branch.branch_name}
                    </CommandItem>
                  ))}
                </CommandGroup>
              </Command>
            </PopoverContent>
          </Popover>
        </div>
      </CardContent>
      {actions}
    </Card>
  );
}
