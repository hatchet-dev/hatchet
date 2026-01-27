import { useWorkflows } from '../../hooks/use-workflows';
import { Combobox } from '@/components/v1/molecules/combobox/combobox';
import { ToolbarType } from '@/components/v1/molecules/data-table/data-table-toolbar';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { Button } from '@/components/v1/ui/button';
import { CodeEditor } from '@/components/v1/ui/code-editor';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Input } from '@/components/v1/ui/input';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, {
  CronWorkflows,
  queries,
  ScheduledWorkflows,
  V1WorkflowRunDetails,
  Workflow,
} from '@/lib/api';
import { formatCron } from '@/lib/cron';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { debounce } from 'lodash';
import { useCallback, useMemo, useState, useEffect } from 'react';
import { BiDownArrowCircle } from 'react-icons/bi';

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
  const queryClient = useQueryClient();
  const { tenantId } = useCurrentTenantId();
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

  const [selectedWorkflowId, setSelectedWorkflowId] = useState(
    defaultWorkflow?.metadata.id,
  );

  const [workflowSearch, setWorkflowSearch] = useState('');
  const [debouncedWorkflowSearch, setDebouncedWorkflowSearch] = useState('');

  const debouncedSetSearch = useMemo(
    () => debounce((value: string) => setDebouncedWorkflowSearch(value), 300),
    [],
  );

  const workflowVersionQuery = useQuery({
    ...queries.workflows.getVersion(selectedWorkflowId || ''),
    enabled: !!selectedWorkflowId,
  });
  const selectedWorkflow = workflowVersionQuery.data;

  const handleSearchChange = useCallback(
    (value: string) => {
      setWorkflowSearch(value);
      debouncedSetSearch(value);
    },
    [debouncedSetSearch],
  );

  useEffect(() => {
    return () => {
      debouncedSetSearch.cancel();
    };
  }, [debouncedSetSearch]);

  const handleClose = useCallback(() => {
    onClose();
    setInput('{}');
    setAddlMeta('{}');
    setErrors([]);
    setSelectedWorkflowId(defaultWorkflow?.metadata.id);
    setTimingOption(defaultTimingOption);
    setScheduleTime(new Date());
    setCronExpression('* * * * *');
    setCronName('');
    setWorkflowSearch('');
    setDebouncedWorkflowSearch('');
    debouncedSetSearch.cancel();
  }, [onClose, defaultWorkflow, defaultTimingOption, debouncedSetSearch]);

  const cronPretty = useMemo(() => {
    try {
      return {
        pretty: formatCron(cronExpression),
      };
    } catch (e) {
      console.error(e);
      return { error: e as string };
    }
  }, [cronExpression]);

  const { handleApiError } = useApiError({
    setErrors,
  });

  const { data: workflowKeys } = useQuery({
    queryKey: [
      'workflow:list',
      tenantId,
      {
        limit: 200,
        name: debouncedWorkflowSearch || undefined,
      },
    ],
    queryFn: async () => {
      const response = await api.workflowList(tenantId, {
        limit: 200,
        name: debouncedWorkflowSearch || undefined,
      });
      return response.data;
    },
    refetchInterval: 15000,
  });

const jsonSchema = selectedWorkflow?.inputJsonSchema;

  const triggerNowMutation = useMutation({
    mutationKey: ['workflow-run:create', selectedWorkflow?.metadata.id],
    mutationFn: async (data: { input: object; addlMeta: object }) => {
      if (!selectedWorkflow) {
        return;
      }

      const res = await api.v1WorkflowRunCreate(tenantId, {
        workflowName: selectedWorkflow.name,
        input: data.input,
        additionalMetadata: data.addlMeta,
      });

      return res.data;
    },
    onMutate: () => {
      setErrors([]);
    },
    onSuccess: (workflowRun: V1WorkflowRunDetails | undefined) => {
      if (!workflowRun) {
        return;
      }

      navigate({
        to: appRoutes.tenantRunRoute.to,
        params: { tenant: tenantId, run: workflowRun.run.metadata.id },
      });
    },
    onError: handleApiError,
  });

  const triggerScheduleMutation = useMutation({
    mutationKey: ['workflow-run:schedule', selectedWorkflow?.metadata.id],
    mutationFn: async (data: {
      input: object;
      addlMeta: object;
      scheduledAt: string;
    }) => {
      if (!selectedWorkflow) {
        return;
      }

      const res = await api.scheduledWorkflowRunCreate(
        tenantId,
        selectedWorkflow?.name,
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
    onSuccess: async (workflowRun: ScheduledWorkflows | undefined) => {
      if (!workflowRun) {
        return;
      }
      handleClose();
      await queryClient.invalidateQueries({
        queryKey: ['scheduledRuns', 'list'],
      });
      navigate({
        to: appRoutes.tenantScheduledRoute.to,
        params: { tenant: tenantId },
      });
    },
    onError: handleApiError,
  });

  const triggerCronMutation = useMutation({
    mutationKey: ['workflow-run:cron', selectedWorkflow?.metadata.id],
    mutationFn: async (data: {
      input: object;
      addlMeta: object;
      cron: string;
      cronName: string;
    }) => {
      if (!selectedWorkflow) {
        return;
      }

      const res = await api.cronWorkflowTriggerCreate(
        tenantId,
        selectedWorkflow?.name,
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
    onSuccess: async (workflowRun: CronWorkflows | undefined) => {
      if (!workflowRun) {
        return;
      }
      handleClose();
      await queryClient.invalidateQueries({
        queryKey: ['cronJobs', 'list'],
      });
      navigate({
        to: appRoutes.tenantCronJobsRoute.to,
        params: { tenant: tenantId },
      });
    },
    onError: handleApiError,
  });

  const handleSubmit = () => {
    if (!selectedWorkflow) {
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

  return (
    <Dialog
      open={show}
      onOpenChange={(open) => {
        if (!open) {
          handleClose();
        }
      }}
    >
      <DialogContent className="max-h-[90%] overflow-auto py-12 sm:max-w-[625px]">
        <DialogHeader className="gap-2">
          <DialogTitle>Trigger Run</DialogTitle>
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
            <BiDownArrowCircle className="mr-2 h-5 w-5 text-gray-700 dark:text-gray-300" />
          }
          searchValue={workflowSearch}
          onSearchChange={handleSearchChange}
          emptyMessage={
            debouncedWorkflowSearch
              ? `No workflows matching "${debouncedWorkflowSearch}"`
              : 'No workflows found'
          }
        />
        <div className="font-bold">Input</div>
        <CodeEditor
          code={input || '{}'}
          setCode={setInput}
          language="json"
          height="180px"
          jsonSchema={jsonSchema}
        />
        <div className="font-bold">Additional Metadata</div>
        <CodeEditor
          code={addlMeta || '{}'}
          setCode={setAddlMeta}
          height="90px"
          language="json"
        />
        <div>
          <div className="mb-2 font-bold">Timing</div>
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
                <div className="mb-2 font-bold">Select Date and Time</div>
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
                <div className="mt-2 flex gap-2">
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
                <div className="mb-2 font-bold">Cron Name</div>
                <Input
                  type="text"
                  value={cronName}
                  onChange={(e) => setCronName(e.target.value)}
                  placeholder="e.g., cron-name"
                  className="mb-2 w-full"
                />
                <div className="mb-2 font-bold">Cron Expression</div>
                <Input
                  type="text"
                  value={cronExpression}
                  onChange={(e) => setCronExpression(e.target.value)}
                  placeholder="e.g., 0 0 * * *"
                  className="w-full"
                />
                <div className="text-sm text-gray-500">
                  {cronPretty?.error || `(runs ${cronPretty?.pretty})`}
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </div>

        <div className="flex justify-end">
          <Button
            disabled={
              triggerNowMutation.isPending ||
              triggerScheduleMutation.isPending ||
              triggerCronMutation.isPending ||
              !selectedWorkflow
            }
            onClick={handleSubmit}
          >
            Submit
          </Button>
        </div>
        {(errors.length > 0 ||
          triggerNowMutation.error ||
          triggerScheduleMutation.error ||
          triggerCronMutation.error) && (
          <div className="mt-4">
            {errors.map((error, index) => (
              <div key={index} className="text-sm text-red-500">
                {error}
              </div>
            ))}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
