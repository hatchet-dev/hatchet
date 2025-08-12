import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import api, {
  CronWorkflows,
  ScheduledWorkflows,
  V1WorkflowRunDetails,
  Workflow,
} from '@/lib/api';
import { useCallback, useMemo, useState } from 'react';
import { Button } from '@/components/v1/ui/button';
import { useApiError } from '@/lib/hooks';
import { useMutation, useInfiniteQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { CodeEditor } from '@/components/v1/ui/code-editor';
import { Input } from '@/components/v1/ui/input';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { BiDownArrowCircle } from 'react-icons/bi';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/v1/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { formatCron } from '@/lib/utils';

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
  }, [onClose, defaultWorkflow, defaultTimingOption]);

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

  // Custom hook to fetch workflows with scroll-based pagination
  const {
    data: workflowData,
    isFetched,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['workflows', 'paginated', tenantId],
    queryFn: async ({ pageParam = 0 }) => {
      const response = await api.workflowList(tenantId, {
        limit: 200,
        offset: pageParam,
      });
      return response.data;
    },
    getNextPageParam: (lastPage) => {
      if (
        lastPage.pagination?.next_page !== lastPage.pagination?.current_page &&
        lastPage.pagination?.next_page !== undefined
      ) {
        return (lastPage.pagination.current_page || 0) * 200;
      }
      return undefined;
    },
    initialPageParam: 0,
    refetchInterval: 15000,
  });

  const handleLoadMore = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const workflowKeys = useMemo(() => {
    if (!workflowData?.pages) {
      return undefined;
    }

    const allRows = workflowData.pages.flatMap((page) => page.rows || []);
    return {
      rows: allRows,
      pagination: workflowData.pages[workflowData.pages.length - 1]?.pagination,
    };
  }, [workflowData]);

  const selectedWorkflow = useMemo(
    () => workflowKeys?.rows?.find((w) => w.metadata.id === selectedWorkflowId),
    [selectedWorkflowId, workflowKeys],
  );

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

      navigate(`/tenants/${tenantId}/runs/${workflowRun.run.metadata.id}`);
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
    onSuccess: (workflowRun: ScheduledWorkflows | undefined) => {
      if (!workflowRun) {
        return;
      }
      handleClose();
      navigate(`/tenants/${tenantId}/scheduled`);
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
    onSuccess: (workflowRun: CronWorkflows | undefined) => {
      if (!workflowRun) {
        return;
      }
      handleClose();
      navigate(`/tenants/${tenantId}/cron-jobs`);
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

  if ((!workflowKeys || workflowKeys.rows?.length === 0) && isFetched) {
    return (
      <Dialog
        open={show}
        onOpenChange={(open) => {
          if (!open) {
            handleClose();
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
      open={show}
      onOpenChange={(open) => {
        if (!open) {
          handleClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-[625px] py-12 max-h-[90%] overflow-auto">
        <DialogHeader className="gap-2">
          <DialogTitle>Trigger Run</DialogTitle>
          <DialogDescription className="text-muted-foreground">
            Trigger a task or workflow to run now, at a scheduled time, or on a
            cron schedule.
          </DialogDescription>
        </DialogHeader>

        <div className="font-bold">Task or Workflow</div>
        <Popover>
          <PopoverTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              className="h-8 border-dashed w-full justify-start"
            >
              <BiDownArrowCircle className="h-5 w-5 text-gray-700 dark:text-gray-300 mr-2" />
              {selectedWorkflow?.name || 'Select Task or Workflow'}
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-[300px] p-2" align="start">
            <Command>
              <CommandInput placeholder="Search workflows..." />
              <CommandList
                className="max-h-60 overflow-y-auto"
                onScroll={(e) => {
                  const { scrollTop, scrollHeight, clientHeight } =
                    e.currentTarget;
                  if (scrollHeight - scrollTop - clientHeight < 50) {
                    handleLoadMore();
                  }
                }}
                onWheel={(e) => {
                  e.currentTarget.scrollTop += e.deltaY;
                }}
              >
                <CommandEmpty>No workflows found.</CommandEmpty>
                <CommandGroup>
                  {workflowKeys?.rows?.map((workflow) => (
                    <CommandItem
                      key={workflow.metadata.id}
                      onSelect={() => {
                        setSelectedWorkflowId(workflow.metadata.id);
                      }}
                      className={
                        selectedWorkflowId === workflow.metadata.id
                          ? 'bg-accent'
                          : ''
                      }
                    >
                      {workflow.name}
                    </CommandItem>
                  ))}
                  {isFetchingNextPage && (
                    <CommandItem disabled>
                      <div className="flex items-center justify-center w-full py-2">
                        Loading more workflows...
                      </div>
                    </CommandItem>
                  )}
                  {!hasNextPage &&
                    workflowKeys?.rows &&
                    workflowKeys.rows.length > 200 && (
                      <CommandItem disabled>
                        <div className="flex items-center justify-center w-full py-1 text-xs text-gray-500">
                          All workflows loaded
                        </div>
                      </CommandItem>
                    )}
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
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
                <div className="font-bold mb-2">Cron Name</div>
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
                  {cronPretty?.error || `(runs ${cronPretty?.pretty})`}
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </div>

        <div className="flex justify-end">
          <Button
            className="w-fit mt-6"
            disabled={
              triggerNowMutation.isPending ||
              triggerScheduleMutation.isPending ||
              triggerCronMutation.isPending ||
              !selectedWorkflow
            }
            onClick={handleSubmit}
          >
            Run Task
          </Button>
        </div>
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
