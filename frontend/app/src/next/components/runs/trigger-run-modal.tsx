import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/next/components/ui/dialog';
import { Button } from '@/next/components/ui/button';
import { Input } from '@/next/components/ui/input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/next/components/ui/select';
import CronPrettifier from 'cronstrue';
import { TimePicker } from '@/next/components/ui/time-picker';
import useDefinitions from '@/next/hooks/use-definitions';
import { useNavigate } from 'react-router-dom';
import { useState, useMemo, useEffect } from 'react';
import {
  V1WorkflowRunDetails,
  Workflow,
  ScheduledWorkflows,
  CronWorkflows,
} from '@/lib/api';
import { RunsProvider, useRuns } from '@/next/hooks/use-runs';
import { CronsProvider, useCrons } from '@/next/hooks/use-crons';
import { SchedulesProvider, useSchedules } from '@/next/hooks/use-schedules';
import { CodeEditor } from '@/components/v1/ui/code-editor';
import { ROUTES } from '@/next/lib/routes';
import { RunDetailProvider, useRunDetail } from '@/next/hooks/use-run-detail';
import { getFriendlyWorkflowRunId } from '@/next/components/runs/run-id';
import { FaCodeBranch } from 'react-icons/fa';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';

type TimingOption = 'now' | 'schedule' | 'cron';

type TriggerRunCapability =
  | 'workflow'
  | 'fromRecent'
  | 'input'
  | 'additionalMeta'
  | 'timing';

type TriggerRunModalProps = {
  show: boolean;
  onClose: () => void;
  defaultTimingOption?: TimingOption;
  defaultInput?: string;
  defaultAddlMeta?: string;
  defaultWorkflowId?: string;
  defaultRunId?: string;
  onRun?: (
    run: V1WorkflowRunDetails | ScheduledWorkflows | CronWorkflows,
  ) => void;
  disabledCapabilities?: TriggerRunCapability[];
};

export function TriggerRunModal(props: TriggerRunModalProps) {
  if (!props.show) {
    return null;
  }

  return (
    <CronsProvider>
      <SchedulesProvider>
        <RunsProvider
          initialPagination={{
            initialPageSize: 5,
          }}
          initialFilters={{
            workflow_ids: props.defaultWorkflowId
              ? [props.defaultWorkflowId]
              : undefined,
          }}
        >
          <TriggerRunModalContent {...props} />
        </RunsProvider>
      </SchedulesProvider>
    </CronsProvider>
  );
}

function WithPreviousInput({
  setInput,
  setAddlMeta,
}: {
  setInput: (input: string) => void;
  setAddlMeta: (addlMeta: string) => void;
}) {
  const { data: selectedRunDetails } = useRunDetail();
  useEffect(() => {
    if (selectedRunDetails?.run) {
      setInput(JSON.stringify(selectedRunDetails.run.input, null, 2));
      setAddlMeta(
        JSON.stringify(selectedRunDetails.run.additionalMetadata, null, 2),
      );
    }
  }, [selectedRunDetails, setAddlMeta, setInput]);

  return null;
}

const defaultDisabledCapabilities: TriggerRunCapability[] = [];

function TriggerRunModalContent({
  show,
  onClose,
  defaultTimingOption = 'now',
  defaultInput = '{}',
  defaultAddlMeta = '{}',
  defaultWorkflowId,
  defaultRunId,
  disabledCapabilities = defaultDisabledCapabilities,
  onRun,
}: TriggerRunModalProps) {
  const navigate = useNavigate();
  const { data: workflows } = useDefinitions();
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<
    string | undefined
  >(defaultWorkflowId);
  const { tenantId } = useCurrentTenantId();

  const { data: recentRuns, triggerNow } = useRuns();

  const [selectedRunId, setSelectedRunId] = useState<string>(
    defaultRunId || '',
  );
  const { create: createCron } = useCrons();
  const { create: createSchedule } = useSchedules();

  const [input, setInput] = useState<string>(defaultInput);
  const [addlMeta, setAddlMeta] = useState<string>(defaultAddlMeta);
  const [errors, setErrors] = useState<string[]>([]);
  const [timingOption, setTimingOption] =
    useState<TimingOption>(defaultTimingOption);
  const [scheduleTime, setScheduleTime] = useState<Date | undefined>(
    new Date(),
  );
  const [cronExpression, setCronExpression] = useState<string>('* * * * *');
  const [cronName, setCronName] = useState<string>('');

  const resetForm = () => {
    setInput(defaultInput);
    setAddlMeta(defaultAddlMeta);
    setErrors([]);
    setTimingOption(defaultTimingOption);
    setScheduleTime(new Date());
    setCronExpression('* * * * *');
    setCronName('');
    setSelectedWorkflowId(defaultWorkflowId);
    setSelectedRunId('');
  };

  const cronPretty = useMemo(() => {
    try {
      return {
        pretty: CronPrettifier.toString(cronExpression).toLowerCase(),
      };
    } catch (e) {
      console.error(e);
      return { error: e as string };
    }
  }, [cronExpression]);

  const selectedWorkflow = useMemo(() => {
    if (!workflows) {
      return undefined;
    }
    return workflows.find(
      (w: Workflow) => w.metadata.id === selectedWorkflowId,
    );
  }, [workflows, selectedWorkflowId]);

  const handleSubmit = () => {
    if (!selectedWorkflow) {
      setErrors(['No workflow selected.']);
      return;
    }

    const inputObj = JSON.parse(input);
    const addlMetaObj = JSON.parse(addlMeta);

    if (timingOption === 'now') {
      triggerNow.mutate(
        {
          workflowName: selectedWorkflow.name,
          input: inputObj,
          additionalMetadata: addlMetaObj,
        },
        {
          onSuccess: (workflowRun) => {
            if (!workflowRun) {
              return;
            }
            onClose();
            if (onRun) {
              onRun(workflowRun);
            } else {
              navigate(
                ROUTES.runs.detail(tenantId, workflowRun.run.metadata.id),
              );
            }
          },
          onError: (error) => {
            setErrors([error.message]);
          },
        },
      );
    } else if (timingOption === 'schedule') {
      if (!scheduleTime) {
        setErrors(['Please select a date and time for scheduling.']);
        return;
      }
      createSchedule.mutate(
        {
          workflowName: selectedWorkflow.name,
          data: {
            input: inputObj,
            additionalMetadata: addlMetaObj,
            triggerAt: new Date(scheduleTime.getTime()).toISOString(),
          },
        },
        {
          onSuccess: (schedule) => {
            onClose();
            if (onRun) {
              onRun(schedule);
            } else {
              navigate(ROUTES.scheduled.list(tenantId));
            }
          },
          onError: (error: any) => {
            if (error?.response?.data?.errors) {
              setErrors(error.response.data.errors);
            } else {
              setErrors([error.message || 'Failed to schedule run']);
            }
          },
        },
      );
    } else if (timingOption === 'cron') {
      if (!cronExpression) {
        setErrors(['Please enter a valid cron expression.']);
        return;
      }
      if (!cronName) {
        setErrors(['Please enter a name for the cron job.']);
        return;
      }
      createCron.mutate(
        {
          workflowId: selectedWorkflow.name,
          data: {
            input: inputObj,
            additionalMetadata: addlMetaObj,
            cronName: cronName,
            cronExpression: cronExpression,
          },
        },
        {
          onSuccess: (cron) => {
            onClose();
            if (onRun) {
              onRun(cron);
            } else {
              navigate(ROUTES.crons.list(tenantId));
            }
          },
          onError: (error: any) => {
            if (error?.response?.data?.errors) {
              setErrors(error.response.data.errors);
            } else {
              setErrors([error.message || 'Failed to create cron job']);
            }
          },
        },
      );
    }
  };

  return (
    <Dialog
      open={show}
      onOpenChange={(open) => {
        if (!open) {
          resetForm();
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-[625px]">
        <DialogHeader>
          <DialogTitle>Trigger Run</DialogTitle>
          <DialogDescription>
            Trigger a workflow to run now, at a scheduled time, or on a cron
            schedule.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {selectedRunId && !disabledCapabilities.includes('fromRecent') ? (
            <RunDetailProvider runId={selectedRunId}>
              <WithPreviousInput
                setInput={setInput}
                setAddlMeta={setAddlMeta}
              />
            </RunDetailProvider>
          ) : null}
          {!disabledCapabilities.includes('workflow') && (
            <div>
              <label className="text-sm font-medium">Workflow</label>
              <Select
                value={selectedWorkflowId}
                onValueChange={(value) => {
                  setSelectedWorkflowId(value);
                  setSelectedRunId('');
                }}
              >
                <SelectTrigger className="w-full mt-1">
                  <SelectValue placeholder="Select a workflow" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="placeholder">Select a workflow</SelectItem>
                  {workflows?.map((workflow: Workflow) => (
                    <SelectItem
                      key={workflow.metadata.id}
                      value={workflow.metadata.id}
                    >
                      {workflow.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          {!disabledCapabilities.includes('fromRecent') && (
            <div>
              <label className="text-sm font-medium flex items-center gap-2">
                <FaCodeBranch className="text-muted-foreground" size={16} />
                From Recent Run
              </label>
              <Select
                value={selectedRunId}
                disabled={!selectedWorkflowId}
                onValueChange={(value) => {
                  setSelectedRunId(value);
                }}
              >
                <SelectTrigger className="w-full mt-1">
                  <SelectValue placeholder="Select a recent run" />
                </SelectTrigger>
                <SelectContent>
                  {(() => {
                    const filteredRuns =
                      recentRuns?.filter(
                        (run) => run.workflowId === selectedWorkflowId,
                      ) || [];

                    if (filteredRuns.length === 0) {
                      return (
                        <SelectItem value="no-runs" disabled>
                          No recent runs available
                        </SelectItem>
                      );
                    }

                    return filteredRuns.map((run) => (
                      <SelectItem key={run.metadata.id} value={run.metadata.id}>
                        {getFriendlyWorkflowRunId(run)}
                      </SelectItem>
                    ));
                  })()}
                </SelectContent>
              </Select>
            </div>
          )}

          {!disabledCapabilities.includes('input') && (
            <div>
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium">Input</label>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setInput('{}');
                    setSelectedRunId('');
                  }}
                  className="h-8 px-2"
                >
                  Clear
                </Button>
              </div>
              <CodeEditor
                language="json"
                className="mt-1"
                height="180px"
                code={input}
                setCode={(code) => code && setInput(code)}
              />
            </div>
          )}

          {!disabledCapabilities.includes('additionalMeta') && (
            <div>
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium">
                  Additional Metadata
                </label>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setAddlMeta('{}')}
                  className="h-8 px-2"
                >
                  Clear
                </Button>
              </div>
              <CodeEditor
                language="json"
                className="mt-1"
                height="90px"
                code={addlMeta}
                setCode={(code) => code && setAddlMeta(code)}
              />
            </div>
          )}

          {!disabledCapabilities.includes('timing') && (
            <div>
              <label className="text-sm font-medium">Timing</label>
              <Tabs
                value={timingOption}
                onValueChange={(value: string) =>
                  setTimingOption(value as TimingOption)
                }
              >
                <TabsList>
                  <TabsTrigger value="now">Now</TabsTrigger>
                  <TabsTrigger value="schedule">Schedule</TabsTrigger>
                  <TabsTrigger value="cron">Cron</TabsTrigger>
                </TabsList>
                <TabsContent value="now" />
                <TabsContent value="schedule">
                  <div className="mt-4">
                    <div className="font-bold mb-2">Select Date and Time</div>
                    <div className="flex gap-2">
                      <TimePicker
                        date={scheduleTime}
                        setDate={setScheduleTime}
                        timezone="Local"
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
                    <div className="text-sm text-muted-foreground mt-1">
                      {cronPretty.error || `(runs ${cronPretty.pretty} UTC)`}
                    </div>
                  </div>
                </TabsContent>
              </Tabs>
            </div>
          )}

          {errors.length > 0 && (
            <div className="text-sm text-destructive">
              {errors.map((error, i) => (
                <p key={i}>{error}</p>
              ))}
            </div>
          )}

          <div className="flex justify-end">
            <Button
              onClick={handleSubmit}
              loading={
                triggerNow.isPending ||
                createSchedule.isPending ||
                createCron.isPending
              }
            >
              {timingOption === 'now'
                ? 'Run Now'
                : timingOption === 'schedule'
                  ? 'Schedule Run'
                  : 'Create Cron'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
