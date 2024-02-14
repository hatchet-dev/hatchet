import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import api, { LinkGithubRepositoryRequest, Workflow, queries } from '@/lib/api';
import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { PlusIcon } from '@heroicons/react/24/outline';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { z } from 'zod';
import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Label } from '@/components/ui/label';

const schema = z.object({
  installation: z.string(),
  repoOwnerName: z.string(),
  branch: z.string(),
});

export function DeploymentSettingsForm({
  workflow,
  show,
  onClose,
}: {
  workflow: Workflow;
  show: boolean;
  onClose: () => void;
}) {
  const [errors, setErrors] = useState<string[]>([]);

  const { handleApiError } = useApiError({
    setErrors,
  });

  const linkGithubMutation = useMutation({
    mutationKey: ['workflow:update:link-github', workflow?.metadata.id],
    mutationFn: async (input: LinkGithubRepositoryRequest) => {
      if (!workflow) {
        return;
      }

      const res = await api.workflowUpdateLinkGithub(
        workflow?.metadata.id,
        input,
      );

      return res.data;
    },
    onMutate: () => {
      setErrors([]);
    },
    onSuccess: () => {
      onClose();
    },
    onError: handleApiError,
  });

  return (
    <Dialog
      open={!!workflow && show}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-[625px] py-12">
        <DialogHeader>
          <DialogTitle>Deployment settings</DialogTitle>
          <DialogDescription>
            You can change the deployment settings of your workflow here.
          </DialogDescription>
        </DialogHeader>
        <InnerForm
          workflow={workflow}
          onSubmit={(opts) => {
            const repoOwner = getRepoOwner(opts.repoOwnerName);
            const repoName = getRepoName(opts.repoOwnerName);

            if (!repoOwner || !repoName) {
              return;
            }

            linkGithubMutation.mutate({
              installationId: opts.installation,
              gitRepoOwner: repoOwner,
              gitRepoName: repoName,
              gitRepoBranch: opts.branch,
            });
          }}
          isLoading={linkGithubMutation.isPending}
          errors={errors}
        />
      </DialogContent>
    </Dialog>
  );
}

interface InnerFormProps {
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  errors?: string[];
  workflow: Workflow;
}

function InnerForm({ workflow, onSubmit, isLoading, errors }: InnerFormProps) {
  const { watch, handleSubmit, control, setValue } = useForm<
    z.infer<typeof schema>
  >({
    resolver: zodResolver(schema),
    defaultValues: {
      installation: workflow.deployment?.githubAppInstallationId,
      repoOwnerName:
        workflow.deployment?.gitRepoOwner &&
        workflow.deployment?.gitRepoName &&
        getRepoOwnerName(
          workflow.deployment?.gitRepoOwner,
          workflow.deployment?.gitRepoName,
        ),
      branch: workflow.deployment?.gitRepoBranch,
    },
  });

  const installation = watch('installation');
  const repoOwnerName = watch('repoOwnerName');
  const branch = watch('branch');

  const listInstallationsQuery = useQuery({
    ...queries.github.listInstallations,
  });

  const listReposQuery = useQuery({
    ...queries.github.listRepos(installation),
  });

  const listBranchesQuery = useQuery({
    ...queries.github.listBranches(
      installation,
      getRepoOwner(repoOwnerName),
      getRepoName(repoOwnerName),
    ),
  });

  useEffect(() => {
    if (
      listInstallationsQuery.isSuccess &&
      listInstallationsQuery.data.rows.length > 0 &&
      !installation
    ) {
      setValue(
        'installation',
        listInstallationsQuery.data?.rows[0]?.metadata.id,
      );
    }
  }, [listInstallationsQuery, setValue, installation]);

  return (
    <>
      <div className="grid gap-4">
        <Label htmlFor="role">Github account</Label>
        <Controller
          control={control}
          name="installation"
          render={({ field }) => {
            return (
              <Select
                onValueChange={(value: string) => {
                  field.onChange(value);
                  setValue('repoOwnerName', '');
                  setValue('branch', '');
                }}
                {...field}
              >
                <SelectTrigger className="w-[180px]">
                  <SelectValue id="role" placeholder="Choose account" />
                </SelectTrigger>
                <SelectContent>
                  {listInstallationsQuery.data?.rows.map((i) => (
                    <SelectItem key={i.metadata.id} value={i.metadata.id}>
                      {i.account_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            );
          }}
        />
        <Label htmlFor="role">Github repo</Label>
        <Controller
          control={control}
          name="repoOwnerName"
          render={({ field }) => {
            return (
              <Select
                onValueChange={(value) => {
                  field.onChange(value);
                  setValue('branch', '');
                }}
                {...field}
              >
                <SelectTrigger className="w-[180px]">
                  <SelectValue id="role" placeholder="Choose repository" />
                </SelectTrigger>
                <SelectContent>
                  {listReposQuery.data?.map((i) => (
                    <SelectItem
                      key={i.repo_owner + i.repo_name}
                      value={getRepoOwnerName(i.repo_owner, i.repo_name)}
                    >
                      {i.repo_owner}/{i.repo_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            );
          }}
        />
        <Label htmlFor="role">Github branch</Label>
        <Controller
          control={control}
          name="branch"
          render={({ field }) => {
            return (
              <Select onValueChange={field.onChange} {...field}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue id="role" placeholder="Choose branch" />
                </SelectTrigger>
                <SelectContent>
                  {listBranchesQuery.data?.map((i) => (
                    <SelectItem key={i.branch_name} value={i.branch_name}>
                      {i.branch_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            );
          }}
        />
        {errors?.map((error, index) => (
          <div key={index} className="text-red-500 text-sm">
            {error}
          </div>
        ))}
        <Button
          disabled={!installation || !repoOwnerName || !branch}
          onClick={handleSubmit(onSubmit)}
        >
          {isLoading && <PlusIcon className="h-4 w-4 animate-spin" />}
          Save
        </Button>
      </div>
    </>
  );
}

function getRepoOwnerName(repoOwner: string, repoName: string) {
  return `${repoOwner}::${repoName}`;
}

function getRepoOwner(repoOwnerName?: string) {
  if (!repoOwnerName) {
    return;
  }

  const splArr = repoOwnerName.split('::');
  if (splArr.length > 1) {
    return splArr[0];
  }
}

function getRepoName(repoOwnerName?: string) {
  if (!repoOwnerName) {
    return;
  }

  const splArr = repoOwnerName.split('::');
  if (splArr.length > 1) {
    return splArr[1];
  }
}
