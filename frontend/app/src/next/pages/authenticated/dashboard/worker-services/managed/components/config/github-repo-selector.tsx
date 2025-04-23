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

export type GithubRepoSelectorValue = Omit<
  CreateManagedWorkerBuildConfigRequest,
  'steps'
>;

interface GithubRepoSelectorProps {
  value: GithubRepoSelectorValue;
  onChange: (value: GithubRepoSelectorValue) => void;
}

export function GithubRepoSelector({
  value,
  onChange,
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

  const selectedInstallationName = installations.data?.find(
    (i) => i.metadata.id === value.githubInstallationId,
  )?.account_name;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Worker Source</CardTitle>
        <CardDescription>
          Select the GitHub repository you want to use for your worker service.
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
            <PopoverContent className="w-full p-0">
              <Command>
                <CommandInput placeholder="Search accounts..." />
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
                disabled={
                  !value.githubRepositoryOwner || !value.githubRepositoryName
                }
              >
                <div className="flex items-center gap-2">
                  <GitBranchIcon className="h-4 w-4" />
                  {value.githubRepositoryBranch ? (
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
    </Card>
  );
}
