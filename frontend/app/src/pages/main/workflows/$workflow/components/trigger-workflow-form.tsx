import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import api, { Workflow, WorkflowRun } from '@/lib/api';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import invariant from 'tiny-invariant';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { PlusIcon } from '@heroicons/react/24/outline';
import { cn } from '@/lib/utils';
import { useNavigate, useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';
import { CodeEditor } from '@/components/ui/code-editor';

export function TriggerWorkflowForm({
  workflow,
  show,
  onClose,
}: {
  workflow: Workflow;
  show: boolean;
  onClose: () => void;
}) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const navigate = useNavigate();

  const [input, setInput] = useState<string | undefined>('{}');
  const [addlMeta, setAddlMeta] = useState<string | undefined>('{}');
  const [errors, setErrors] = useState<string[]>([]);

  const { handleApiError } = useApiError({
    setErrors,
  });

  const triggerWorkflowMutation = useMutation({
    mutationKey: ['workflow-run:create', workflow?.metadata.id],
    mutationFn: async (data: { input: object; addlMeta: object }) => {
      if (!workflow) {
        return;
      }

      const res = await api.workflowRunCreate(workflow?.metadata.id, {
        input: data.input,
        additionalMetadata: data.addlMeta,
      });

      return res.data;
    },
    onMutate: () => {
      setErrors([]);
    },
    onSuccess: (workflowRun: WorkflowRun | undefined) => {
      if (!workflowRun) {
        return;
      }

      navigate(`/workflow-runs/${workflowRun.metadata.id}`);
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
          <DialogTitle>Trigger this workflow</DialogTitle>
          <DialogDescription>
            You can change the input to your workflow here.
          </DialogDescription>
        </DialogHeader>
        <div className="font-bold">Input</div>
        <CodeEditor
          code={input || '{}'}
          setCode={setInput}
          language="json"
          height="180px"
        />
        <div className="font-bold">Additional Metadata</div>
        <CodeEditor
          code={addlMeta || '{}'}
          setCode={setAddlMeta}
          height="90px"
          language="json"
        />
        <Button
          className="w-fit"
          disabled={triggerWorkflowMutation.isPending}
          onClick={() => {
            const inputObj = JSON.parse(input || '{}');
            const addlMetaObj = JSON.parse(addlMeta || '{}');
            triggerWorkflowMutation.mutate({
              input: inputObj,
              addlMeta: addlMetaObj,
            });
          }}
        >
          <PlusIcon
            className={cn(
              triggerWorkflowMutation.isPending ? 'rotate-180' : '',
              'h-4 w-4 mr-2',
            )}
          />
          Trigger workflow
        </Button>
        {errors.length > 0 && (
          <div className="mt-4">
            {errors.map((error, index) => (
              <div key={index} className="text-red-500 text-sm">
                {error}
              </div>
            ))}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
