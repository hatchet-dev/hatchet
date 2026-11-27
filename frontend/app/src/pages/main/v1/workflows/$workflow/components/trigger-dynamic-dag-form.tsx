import { Combobox } from '@/components/v1/molecules/combobox/combobox';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { Button } from '@/components/v1/ui/button';
import { CodeEditor } from '@/components/v1/ui/code-editor';
import { Input } from '@/components/v1/ui/input';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { V1DynamicDagNode } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { BiDownArrowCircle } from 'react-icons/bi';

export type DynamicDag = {
  name: string;
  nodes: V1DynamicDagNode[];
};

export function TriggerDynamicDagForm({
  defaultDynamicDag,
  defaultInput = '{}',
}: {
  defaultDynamicDag?: DynamicDag;
  defaultInput?: string;
}) {
  const { tenantId } = useCurrentTenantId();
  const navigate = useNavigate();

  const [dagName, setDagName] = useState(
    defaultDynamicDag?.name ?? 'dynamic-dag',
  );
  const [taskNames, setTaskNames] = useState<string[]>(
    defaultDynamicDag?.nodes.map((node) => node.taskName) ?? [],
  );
  const [parentTaskNamesByTaskName, setParentTaskNamesByTaskName] = useState<
    Record<string, string[]>
  >(
    Object.fromEntries(
      defaultDynamicDag?.nodes.map((node) => [
        node.taskName,
        node.parentTaskNames ?? [],
      ]) ?? [],
    ),
  );
  const [input, setInput] = useState(defaultInput);
  const [errors, setErrors] = useState<string[]>([]);

  const { handleApiError } = useApiError({ setErrors });

  const { data: workflows } = useQuery({
    queryKey: ['workflow:list', tenantId, { limit: 200 }],
    queryFn: async () =>
      (await api.workflowList(tenantId, { limit: 200 })).data,
  });

  const selectedParentTaskNames = (taskName: string) =>
    (parentTaskNamesByTaskName[taskName] ?? []).filter((parent) =>
      taskNames.includes(parent),
    );

  const taskOptions =
    workflows?.rows?.map((w) => ({ value: w.name, label: w.name })) ?? [];

  const triggerMutation = useMutation({
    mutationKey: ['workflow-run:create-dynamic-dag', dagName],
    mutationFn: async (data: {
      input: object;
      dynamicDagNodes: V1DynamicDagNode[];
    }) => {
      const res = await api.v1WorkflowRunCreate(tenantId, {
        workflowName: dagName,
        input: data.input,
        dynamicDagNodes: data.dynamicDagNodes,
        return_only_id: true,
      });

      return res.data;
    },
    onMutate: () => setErrors([]),
    onSuccess: (workflowRun) => {
      navigate({
        to: appRoutes.tenantRunRoute.to,
        params: { tenant: tenantId, run: workflowRun.run.metadata.id },
        search: (prev) => ({ ...prev, wasRedirectedFromTrigger: true }),
      });
    },
    onError: handleApiError,
  });

  const handleSubmit = () => {
    let inputObj: object;

    try {
      inputObj = JSON.parse(input || '{}');
    } catch {
      setErrors(['Input must be valid JSON.']);
      return;
    }

    const dynamicDagNodes = taskNames.map((taskName) => ({
      taskName,
      parentTaskNames: selectedParentTaskNames(taskName),
    }));

    triggerMutation.mutate({ input: inputObj, dynamicDagNodes });
  };

  return (
    <>
      <div className="font-bold">DAG Name</div>
      <Input
        value={dagName}
        onChange={(e) => setDagName(e.target.value)}
        placeholder="e.g. my-dynamic-dag"
      />

      <div className="font-bold">Tasks</div>
      <Combobox
        values={taskNames}
        setValues={(values) => setTaskNames([...values])}
        title="Select Tasks"
        options={taskOptions}
        type={ToolbarType.Checkbox}
        icon={
          <BiDownArrowCircle className="mr-2 h-5 w-5 text-gray-700 dark:text-gray-300" />
        }
        emptyMessage="No tasks found"
      />

      {taskNames.length > 0 && (
        <div className="flex flex-col gap-2">
          <div className="font-bold">Dependencies</div>
          {taskNames.map((taskName) => (
            <div
              key={taskName}
              className="flex items-center justify-between gap-4 rounded-md border px-3 py-2"
            >
              <span className="font-mono text-sm">{taskName}</span>
              <Combobox
                values={selectedParentTaskNames(taskName)}
                setValues={(parents) =>
                  setParentTaskNamesByTaskName((prev) => ({
                    ...prev,
                    [taskName]: parents,
                  }))
                }
                title="Runs after"
                options={taskNames
                  .filter((other) => other !== taskName)
                  .map((other) => ({ value: other, label: other }))}
                type={ToolbarType.Checkbox}
                emptyMessage="Select more tasks to add dependencies"
              />
            </div>
          ))}
        </div>
      )}

      <div className="font-bold">Input</div>
      <CodeEditor
        code={input}
        setCode={(code) => setInput(code ?? '{}')}
        language="json"
        height="180px"
      />

      <div className="flex justify-end">
        <Button
          disabled={
            triggerMutation.isPending || taskNames.length === 0 || !dagName
          }
          onClick={handleSubmit}
        >
          Trigger DAG
        </Button>
      </div>

      {errors.length > 0 && (
        <div className="mt-4">
          {errors.map((error, index) => (
            <div key={index} className="text-sm text-red-500">
              {error}
            </div>
          ))}
        </div>
      )}
    </>
  );
}
