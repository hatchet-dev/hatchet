import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import api, {
  CronWorkflows,
  queries,
  ScheduledWorkflows,
  Workflow,
  WorkflowRun,
} from '@/lib/api';
import { useMemo, useState } from 'react';
import { Button } from '@/components/v1/ui/button';
import invariant from 'tiny-invariant';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { PlusIcon } from '@heroicons/react/24/outline';
import { cn } from '@/lib/utils';
import { useNavigate, useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';
import { CodeEditor } from '@/components/v1/ui/code-editor';
import { Input } from '@/components/v1/ui/input';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import CronPrettifier from 'cronstrue';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { BiDownArrowCircle } from 'react-icons/bi';
import { Combobox } from '@/components/v1/molecules/combobox/combobox';

type TimingOption = 'now' | 'schedule' | 'cron';

export function TriggerWorkflowForm({
  defaultWorkflow,
  show,
  onClose,
  defaultTimingOption = 'now',
}: {
  defaultWorkflow?: Workflow;
  show: boolean;
  onClose: () => void;
  defaultTimingOption?: TimingOption;
}) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const navigate = useNavigate();

  const [input, setInput] = useState<string | undefined>('{}');
  const [addlMeta, setAddlMeta] = useState<string | undefined>('{}');
  const [errors, setErrors] = useState<string[]>([]);

  const [timingOption, setTimingOption] =
    useState<TimingOption>(defaultTimingOption);
  const [scheduleTime, setScheduleTime] = useState<Date | undefined>(
    new Date(),
  );
  const [cronExpression, setCronExpression] = useState<string>('* * * * *');
  const [cronName, setCronName] = useState<string>('');

  const [selectedWorkflowId, setSelectedWorkflowId] = useState<
    string | undefined
  >(defaultWorkflow?.metadata.id);

  const cronPretty = useMemo(() => {
    try {
      return {
        pretty: CronPrettifier.toString(cronExpression || '').toLowerCase(),
      };
    } catch (e) {
      console.error(e);
      return { error: e as string };
    }
  }, [cronExpression]);

  const { handleApiError } = useApiError({
    setErrors,
  });

  const { data: workflowKeys, isFetched } = useQuery({
    ...queries.workflows.list(tenant.metadata.id, { limit: 200 }),
  });

  const workflow = useMemo(() => {
    if (defaultWorkflow) {
      return workflowKeys?.rows?.find(
        (w) => w.metadata.id === defaultWorkflow.metadata.id,
      );
    }

    if (selectedWorkflowId) {
      return workflowKeys?.rows?.find(
        (w) => w.metadata.id === selectedWorkflowId,
      );
    }

    return (workflowKeys?.rows || [])[0];
  }, [workflowKeys, defaultWorkflow, selectedWorkflowId]);

  const triggerNowMutation = useMutation({
    mutationKey: ['workflow-run:create', workflow?.metadata.id],
    mutationFn: async (data: { input: object; addlMeta: object }) => {
      if (!workflow) {
        return;
      }

      const res = await api.workflowRunCreate(workflow.metadata.id, {
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

  const triggerScheduleMutation = useMutation({
    mutationKey: ['workflow-run:schedule', workflow?.metadata.id],
    mutationFn: async (data: {
      input: object;
      addlMeta: object;
      scheduledAt: string;
    }) => {
      if (!workflow) {
        return;
      }

      const res = await api.scheduledWorkflowRunCreate(
        tenant.metadata.id,
        workflow?.name,
        {
          input: data.input,
          additionalMetadata: data.addlMeta,
          triggerAt: data.scheduledAt,
        },
      );

      return res.data;
    },
    onMutate: () => {
      setErrors([]);
    },
    onSuccess: (workflowRun: ScheduledWorkflows | undefined) => {
      if (!workflowRun) {
        return;
      }
      onClose();
      navigate(`/scheduled`);
    },
    onError: handleApiError,
  });

  const triggerCronMutation = useMutation({
    mutationKey: ['workflow-run:cron', workflow?.metadata.id],
    mutationFn: async (data: {
      input: object;
      addlMeta: object;
      cron: string;
      cronName: string;
    }) => {
      if (!workflow) {
        return;
      }

      const res = await api.cronWorkflowTriggerCreate(
        tenant.metadata.id,
        workflow?.name,
        {
          input: data.input,
          additionalMetadata: data.addlMeta,
          cronName: data.cronName,
          cronExpression: data.cron,
        },
      );

      return res.data;
    },
    onMutate: () => {
      setErrors([]);
    },
    onSuccess: (workflowRun: CronWorkflows | undefined) => {
      if (!workflowRun) {
        return;
      }
      onClose();
      navigate(`/cron-jobs`);
    },
    onError: handleApiError,
  });

  const handleSubmit = () => {
    if (!workflow) {
      setErrors(['No workflow selected.']);
      return;
    }

    const inputObj = JSON.parse(input || '{}');
    const addlMetaObj = JSON.parse(addlMeta || '{}');

    if (timingOption === 'now') {
      triggerNowMutation.mutate({
        input: inputObj,
        addlMeta: addlMetaObj,
      });
    } else if (timingOption === 'schedule') {
      if (!scheduleTime) {
        setErrors(['Please select a date and time for scheduling.']);
        return;
      }
      triggerScheduleMutation.mutate({
        input: inputObj,
        addlMeta: addlMetaObj,
        scheduledAt: scheduleTime.toISOString(),
      });
    } else if (timingOption === 'cron') {
      if (!cronExpression) {
        setErrors(['Please enter a valid cron expression.']);
        return;
      }
      triggerCronMutation.mutate({
        input: inputObj,
        addlMeta: addlMetaObj,
        cron: cronExpression,
        cronName: cronName,
      });
    }
  };

  if (!workflow && isFetched) {
    return (
      <Dialog
        open={show}
        onOpenChange={(open) => {
          if (!open) {
            onClose();
          }
        }}
      >
        <DialogContent className="sm:max-w-[625px] py-12 max-h-screen overflow-auto">
          <DialogHeader className="gap-2">
            <DialogTitle>Trigger Workflow</DialogTitle>
            <DialogDescription className="text-muted-foreground">
              No workflows found. Create a workflow first.
            </DialogDescription>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog
      open={!!workflow && show}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-[625px] py-12 max-h-screen overflow-auto">
        <DialogHeader className="gap-2">
          <DialogTitle>Trigger Workflow</DialogTitle>
          <DialogDescription className="text-muted-foreground">
            Trigger a workflow to run now, at a scheduled time, or on a cron
            schedule.
          </DialogDescription>
        </DialogHeader>

        <div className="font-bold">Workflow</div>
        <Combobox
          values={selectedWorkflowId ? [selectedWorkflowId] : []}
          setValues={(values) => setSelectedWorkflowId(values[0])}
          title="Select Workflow"
          options={workflowKeys?.rows?.map((w) => ({
            value: w.metadata.id,
            label: w.name,
          }))}
          type={ToolbarType.Radio}
          icon={
            <BiDownArrowCircle className="h-5 w-5 text-gray-700 dark:text-gray-300 mr-2" />
          }
        />
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
        <div>
          <div className="font-bold mb-2">Timing</div>
          <Tabs
            defaultValue={timingOption}
            onValueChange={(value) =>
              setTimingOption(value as 'now' | 'schedule' | 'cron')
            }
          >
            <TabsList>
              <TabsTrigger value="now">Now</TabsTrigger>
              <TabsTrigger value="schedule">Schedule</TabsTrigger>
              <TabsTrigger value="cron">Cron</TabsTrigger>
            </TabsList>
            <TabsContent value="now"></TabsContent>
            <TabsContent value="schedule">
              <div className="mt-4">
                <div className="font-bold mb-2">Select Date and Time</div>
                <div className="flex gap-2">
                  <DateTimePicker
                    date={scheduleTime}
                    setDate={setScheduleTime}
                    label="Trigger at"
                  />
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setScheduleTime(new Date())}
                  >
                    Now
                  </Button>
                </div>
                <div className="flex gap-2 mt-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      const newTime = new Date(scheduleTime || new Date());
                      newTime.setSeconds(newTime.getSeconds() + 15);
                      setScheduleTime(newTime);
                    }}
                  >
                    +15s
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      const newTime = new Date(scheduleTime || new Date());
                      newTime.setMinutes(newTime.getMinutes() + 1);
                      setScheduleTime(newTime);
                    }}
                  >
                    +1m
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      const newTime = new Date(scheduleTime || new Date());
                      newTime.setMinutes(newTime.getMinutes() + 5);
                      setScheduleTime(newTime);
                    }}
                  >
                    +5m
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      const newTime = new Date(scheduleTime || new Date());
                      newTime.setMinutes(newTime.getMinutes() + 15);
                      setScheduleTime(newTime);
                    }}
                  >
                    +15m
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      const newTime = new Date(scheduleTime || new Date());
                      newTime.setMinutes(newTime.getMinutes() + 60);
                      setScheduleTime(newTime);
                    }}
                  >
                    +60m
                  </Button>
                </div>
              </div>
            </TabsContent>
            <TabsContent value="cron">
              <div className="mt-4">
                <div className="font-bold mb-2">Cron Expression</div>
                <Input
                  type="text"
                  value={cronName}
                  onChange={(e) => setCronName(e.target.value)}
                  placeholder="e.g., cron-name"
                  className="w-full mb-2"
                />
                <div className="font-bold mb-2">Cron Expression</div>
                <Input
                  type="text"
                  value={cronExpression}
                  onChange={(e) => setCronExpression(e.target.value)}
                  placeholder="e.g., 0 0 * * *"
                  className="w-full"
                />
                <div className="text-sm text-gray-500">
                  {cronPretty?.error || `(runs ${cronPretty?.pretty} UTC)`}
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </div>

        <Button
          className="w-fit mt-6"
          disabled={
            triggerNowMutation.isPending ||
            triggerScheduleMutation.isPending ||
            triggerCronMutation.isPending
          }
          onClick={handleSubmit}
        >
          <PlusIcon
            className={cn(
              triggerNowMutation.isPending ||
                triggerScheduleMutation.isPending ||
                triggerCronMutation.isPending
                ? 'rotate-180'
                : '',
              'h-4 w-4 mr-2',
            )}
          />
          Trigger workflow
        </Button>
        {(errors.length > 0 ||
          triggerNowMutation.error ||
          triggerScheduleMutation.error ||
          triggerCronMutation.error) && (
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
