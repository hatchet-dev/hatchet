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
import CronPrettifier from 'cronstrue';
import { TimePicker } from '@/next/components/ui/time-picker';
import useDefinitions from '@/next/hooks/use-definitions';
import { useNavigate } from 'react-router-dom';
import { useState, useMemo, useEffect } from 'react';
import { Workflow } from '@/next/lib/api';
import { useRuns } from '@/next/hooks/use-runs';
import useCrons from '@/next/hooks/use-crons';
import useSchedules from '@/next/hooks/use-schedules';
import { CodeEditor } from '@/components/v1/ui/code-editor';
import { ROUTES } from '@/next/lib/routes';
import { useRunDetail } from '@/next/hooks/use-run-detail';
import { getFriendlyWorkflowRunId } from '@/next/components/runs/run-id';
import { PaginationManagerNoOp } from '@/next/hooks/use-pagination';
import { FaCodeBranch } from 'react-icons/fa';

type TimingOption = 'now' | 'schedule' | 'cron';

export function TriggerRunModal({
  show,
  onClose,
  defaultTimingOption = 'now',
  defaultInput = '{}',
  defaultAddlMeta = '{}',
  defaultWorkflowId,
  defaultRunId,
}: {
  show: boolean;
  onClose: () => void;
  defaultTimingOption?: TimingOption;
  defaultInput?: string;
  defaultAddlMeta?: string;
  defaultWorkflowId?: string;
  defaultRunId?: string;
}) {
  const navigate = useNavigate();
  const { data: workflows } = useDefinitions();
  const { data: recentRuns } = useRuns({
    pagination: {
      ...PaginationManagerNoOp,
      pageSize: 5,
    },
  });
  const [selectedRunId, setSelectedRunId] = useState<string>(
    defaultRunId || '',
  );
  const { data: selectedRunDetails } = useRunDetail(selectedRunId);
  const { triggerNow } = useRuns({});
  const { create: createCron } = useCrons({});
  const { create: createSchedule } = useSchedules({});

  useEffect(() => {
    if (selectedRunDetails?.run) {
      setInput(
        JSON.stringify((selectedRunDetails.run.input as any).input, null, 2),
      );
      setAddlMeta(
        JSON.stringify(selectedRunDetails.run.additionalMetadata, null, 2),
      );
    }
  }, [selectedRunDetails]);

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
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<
    string | undefined
  >(defaultWorkflowId);

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
            navigate(ROUTES.runs.detail(workflowRun.run.metadata.id));
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
            triggerAt: new Date(
              scheduleTime.getTime() - scheduleTime.getTimezoneOffset() * 60000,
            ).toISOString(),
          },
        },
        {
          onSuccess: () => {
            onClose();
            navigate(ROUTES.scheduled.list);
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
          onSuccess: () => {
            onClose();
            navigate(ROUTES.crons.list);
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
          <div>
            <label className="text-sm font-medium">Workflow</label>
            <select
              className="w-full mt-1 rounded-md border border-input bg-background px-3 py-2"
              value={selectedWorkflowId}
              onChange={(e) => {
                setSelectedWorkflowId(e.target.value);
                setSelectedRunId(''); // Reset selected run when workflow changes
              }}
            >
              <option value="">Select a workflow</option>
              {workflows?.map((workflow: Workflow) => (
                <option key={workflow.metadata.id} value={workflow.metadata.id}>
                  {workflow.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="text-sm font-medium flex items-center gap-2">
              <FaCodeBranch className="text-muted-foreground" size={16} />
              From Recent Run
            </label>
            <select
              className="w-full mt-1 rounded-md border border-input bg-background px-3 py-2 disabled:opacity-50 disabled:cursor-not-allowed"
              value={selectedRunId}
              disabled={!selectedWorkflowId}
              onChange={(e) => {
                const runId = e.target.value;
                setSelectedRunId(runId);
              }}
            >
              <option value="">Select a recent run</option>
              {recentRuns
                ?.filter((run) => run.workflowId === selectedWorkflowId)
                .map((run) => (
                  <option key={run.metadata.id} value={run.metadata.id}>
                    {getFriendlyWorkflowRunId(run)}
                  </option>
                ))}
            </select>
          </div>

          <div>
            <label className="text-sm font-medium">Input</label>
            <CodeEditor
              language="json"
              className="mt-1"
              height="180px"
              code={input}
              setCode={(code) => code && setInput(code)}
            />
          </div>

          <div>
            <label className="text-sm font-medium">Additional Metadata</label>
            <CodeEditor
              language="json"
              className="mt-1"
              height="90px"
              code={addlMeta}
              setCode={(code) => code && setAddlMeta(code)}
            />
          </div>

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
