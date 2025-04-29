import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package main\n\nimport (\n\t'context'\n\t'fmt'\n\t'time'\n\n\tv1_workflows 'github.com/hatchet-dev/hatchet/examples/go/workflows'\n\t'github.com/hatchet-dev/hatchet/pkg/client'\n\tv1 'github.com/hatchet-dev/hatchet/pkg/v1'\n\t'github.com/joho/godotenv'\n)\n\nfunc priority() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tctx := context.Background()\n\n\tpriorityWorkflow := v1_workflows.Priority(hatchet)\n\n\t// > Running a Task with Priority\n\tpriority := int32(3)\n\n\trunId, err := priorityWorkflow.RunNoWait(ctx, v1_workflows.PriorityInput{\n\t\tUserId: '1234',\n\t}, client.WithPriority(priority))\n\t\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Println(runId)\n\n\t// > Schedule and cron\n\tschedulePriority := int32(3)\n\trunAt := time.Now().Add(time.Minute)\n\n\tscheduledRunId, _ := priorityWorkflow.Schedule(ctx, runAt, v1_workflows.PriorityInput{\n\t\tUserId: '1234',\n\t}, client.WithPriority(schedulePriority))\n\n\tcronId, _ := priorityWorkflow.Cron(ctx, 'my-cron', '* * * * *', v1_workflows.PriorityInput{\n\t\tUserId: '1234',\n\t}, client.WithPriority(schedulePriority))\n\t\n\n\tfmt.Println(scheduledRunId)\n\tfmt.Println(cronId)\n\n\t\n}\n",
  source: 'out/go/run/priority.go',
  blocks: {
    running_a_task_with_priority: {
      start: 31,
      stop: 35,
    },
    schedule_and_cron: {
      start: 44,
      stop: 53,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
