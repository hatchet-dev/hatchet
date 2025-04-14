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
import { useState, useMemo } from 'react';
import { Workflow } from '@/next/lib/api';
import { useRuns } from '@/next/hooks/use-runs';
import useCrons from '@/next/hooks/use-crons';
import useSchedules from '@/next/hooks/use-schedules';
import { CodeEditor } from '@/components/v1/ui/code-editor';
import { ROUTES } from '@/next/lib/routes';

type TimingOption = 'now' | 'schedule' | 'cron';

export function TriggerRunModal({
  show,
  onClose,
  defaultTimingOption = 'now',
  defaultInput = '{}',
  defaultAddlMeta = '{}',
  defaultWorkflowId,
}: {
  show: boolean;
  onClose: () => void;
  defaultTimingOption?: TimingOption;
  defaultInput?: string;
  defaultAddlMeta?: string;
  defaultWorkflowId?: string;
}) {
  const navigate = useNavigate();
  const { data: workflows } = useDefinitions();
  const { triggerNow } = useRuns({});
  const { create: createCron } = useCrons({});
  const { create: createSchedule } = useSchedules({});

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
            triggerAt: scheduleTime.toISOString(),
          },
        },
        {
          onSuccess: (scheduledWorkflow) => {
            if (!scheduledWorkflow.workflowRunId) {
              return;
            }
            onClose();
            navigate(ROUTES.runs.detail(scheduledWorkflow.workflowRunId));
          },
          onError: (error) => {
            setErrors([error.message]);
          },
        },
      );
    } else if (timingOption === 'cron') {
      if (!cronExpression) {
        setErrors(['Please enter a valid cron expression.']);
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
          onError: (error) => {
            setErrors([error.message]);
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
              onChange={(e) => setSelectedWorkflowId(e.target.value)}
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
                  <TimePicker date={scheduleTime} setDate={setScheduleTime} />
                </div>
              </TabsContent>
              <TabsContent value="cron">
                <div className="mt-4 space-y-4">
                  <div>
                    <label className="text-sm font-medium">Cron Name</label>
                    <Input
                      value={cronName}
                      onChange={(e) => setCronName(e.target.value)}
                      placeholder="Enter a name for this cron job"
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium">
                      Cron Expression
                    </label>
                    <Input
                      value={cronExpression}
                      onChange={(e) => setCronExpression(e.target.value)}
                      placeholder="* * * * *"
                    />
                    <p className="text-sm text-muted-foreground mt-1">
                      {cronPretty.error || `(runs ${cronPretty.pretty})`}
                    </p>
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
              Trigger Run
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
