import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import api, {
  CreatePullRequestFromStepRun,
  StepRun,
  StepRunDiff,
  Workflow,
  queries,
} from '@/lib/api';
import { useState } from 'react';
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
import { DiffCodeEditor } from '@/components/ui/code-editor';

const schema = z.object({
  branch: z.string(),
});

export function CreatePRDialog({
  stepRun,
  workflowId,
  diffs,
  show,
  onClose,
}: {
  stepRun: StepRun;
  workflowId: string;
  diffs: StepRunDiff[];
  show: boolean;
  onClose: () => void;
}) {
  const [errors, setErrors] = useState<string[]>([]);

  const { handleApiError } = useApiError({
    setErrors,
  });

  const getWorkflowQuery = useQuery({
    ...queries.workflows.get(workflowId),
  });

  const createPRMutation = useMutation({
    mutationKey: ['step-run:update:create-pr', stepRun.metadata.id],
    mutationFn: async (input: CreatePullRequestFromStepRun) => {
      if (!stepRun) {
        return;
      }

      const res = await api.stepRunUpdateCreatePr(stepRun.metadata.id, input);

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

  const workflow = getWorkflowQuery.data;

  return (
    <Dialog
      open={!!stepRun && show}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-[625px] py-12">
        <DialogHeader>
          <DialogTitle>Create a Pull Request</DialogTitle>
          <DialogDescription>
            Create a pull request to update {stepRun.step?.readableId}.
          </DialogDescription>
        </DialogHeader>
        {workflow && (
          <InnerForm
            workflow={workflow}
            onSubmit={(opts) => {
              if (!opts.branch) {
                return;
              }

              createPRMutation.mutate({
                branchName: opts.branch,
              });
            }}
            isLoading={createPRMutation.isPending}
            errors={errors}
            diffs={diffs}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

interface InnerFormProps {
  onSubmit: (opts: z.infer<typeof schema>) => void;
  isLoading: boolean;
  errors?: string[];
  workflow: Workflow;
  diffs: StepRunDiff[];
}

function InnerForm({
  workflow,
  onSubmit,
  isLoading,
  errors,
  diffs,
}: InnerFormProps) {
  const { watch, handleSubmit, control, setValue } = useForm<
    z.infer<typeof schema>
  >({
    resolver: zodResolver(schema),
    defaultValues: {
      branch: workflow.deployment?.gitRepoBranch,
    },
  });

  const branch = watch('branch');

  const listBranchesQuery = useQuery({
    ...queries.github.listBranches(
      workflow.deployment?.githubAppInstallationId,
      workflow.deployment?.gitRepoOwner,
      workflow.deployment?.gitRepoName,
    ),
  });

  return (
    <>
      <div className="grid gap-4">
        <Label htmlFor="role">
          Select the base branch for the pull request
        </Label>
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
        <Label>Changed values</Label>
        {diffs.map((diff) => (
          <>
            <div className="text-muted-foreground text-sm">{diff.key}</div>
            <DiffCodeEditor
              code={diff.modified}
              originalValue={diff.original}
              language="text"
              height="120px"
            />
          </>
        ))}
        <Button
          disabled={!branch || !listBranchesQuery.data}
          onClick={handleSubmit(onSubmit)}
        >
          {isLoading && <PlusIcon className="h-4 w-4 animate-spin" />}
          Create PR
        </Button>
      </div>
    </>
  );
}
