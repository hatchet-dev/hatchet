import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package v1_workflows\n\nimport (\n\t\'math/rand\'\n\t\'time\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client/create\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/factory\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/workflow\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker/condition\'\n)\n\n// StepOutput represents the output of most tasks in this workflow\ntype StepOutput struct {\n\tRandomNumber int `json:\'randomNumber\'`\n}\n\n// RandomSum represents the output of the sum task\ntype RandomSum struct {\n\tSum int `json:\'sum\'`\n}\n\n// TaskConditionWorkflowResult represents the aggregate output of all tasks\ntype TaskConditionWorkflowResult struct {\n\tStart        StepOutput `json:\'start\'`\n\tWaitForSleep StepOutput `json:\'waitForSleep\'`\n\tWaitForEvent StepOutput `json:\'waitForEvent\'`\n\tSkipOnEvent  StepOutput `json:\'skipOnEvent\'`\n\tLeftBranch   StepOutput `json:\'leftBranch\'`\n\tRightBranch  StepOutput `json:\'rightBranch\'`\n\tSum          RandomSum  `json:\'sum\'`\n}\n\n// taskOpts is a type alias for workflow task options\ntype taskOpts = create.WorkflowTask[struct{}, TaskConditionWorkflowResult]\n\nfunc TaskConditionWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[struct{}, TaskConditionWorkflowResult] {\n\t// > Create a workflow\n\twf := factory.NewWorkflow[struct{}, TaskConditionWorkflowResult](\n\t\tcreate.WorkflowCreateOpts[struct{}]{\n\t\t\tName: \'TaskConditionWorkflow\',\n\t\t},\n\t\thatchet,\n\t)\n\n\t// > Add base task\n\tstart := wf.Task(\n\t\ttaskOpts{\n\t\t\tName: \'start\',\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\t// > Add wait for sleep\n\twaitForSleep := wf.Task(\n\t\ttaskOpts{\n\t\t\tName:    \'waitForSleep\',\n\t\t\tParents: []create.NamedTask{start},\n\t\t\tWaitFor: condition.SleepCondition(time.Second * 10),\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\t// > Add skip on event\n\tskipOnEvent := wf.Task(\n\t\ttaskOpts{\n\t\t\tName:    \'skipOnEvent\',\n\t\t\tParents: []create.NamedTask{start},\n\t\t\tWaitFor: condition.SleepCondition(time.Second * 30),\n\t\t\tSkipIf:  condition.UserEventCondition(\'skip_on_event:skip\', \'true\'),\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\t// > Add branching\n\tleftBranch := wf.Task(\n\t\ttaskOpts{\n\t\t\tName:    \'leftBranch\',\n\t\t\tParents: []create.NamedTask{waitForSleep},\n\t\t\tSkipIf:  condition.ParentCondition(waitForSleep, \'output.randomNumber > 50\'),\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\trightBranch := wf.Task(\n\t\ttaskOpts{\n\t\t\tName:    \'rightBranch\',\n\t\t\tParents: []create.NamedTask{waitForSleep},\n\t\t\tSkipIf:  condition.ParentCondition(waitForSleep, \'output.randomNumber <= 50\'),\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\t// > Add wait for event\n\twaitForEvent := wf.Task(\n\t\ttaskOpts{\n\t\t\tName:    \'waitForEvent\',\n\t\t\tParents: []create.NamedTask{start},\n\t\t\tWaitFor: condition.Or(\n\t\t\t\tcondition.SleepCondition(time.Minute),\n\t\t\t\tcondition.UserEventCondition(\'wait_for_event:start\', \'true\'),\n\t\t\t),\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\treturn &StepOutput{\n\t\t\t\tRandomNumber: rand.Intn(100) + 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\t// > Add sum\n\twf.Task(\n\t\ttaskOpts{\n\t\t\tName: \'sum\',\n\t\t\tParents: []create.NamedTask{\n\t\t\t\tstart,\n\t\t\t\twaitForSleep,\n\t\t\t\twaitForEvent,\n\t\t\t\tskipOnEvent,\n\t\t\t\tleftBranch,\n\t\t\t\trightBranch,\n\t\t\t},\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {\n\t\t\tvar startOutput StepOutput\n\t\t\tif err := ctx.ParentOutput(start, &startOutput); err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\tvar waitForSleepOutput StepOutput\n\t\t\tif err := ctx.ParentOutput(waitForSleep, &waitForSleepOutput); err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\tvar waitForEventOutput StepOutput\n\t\t\tctx.ParentOutput(waitForEvent, &waitForEventOutput)\n\n\t\t\t// Handle potentially skipped tasks\n\t\t\tvar skipOnEventOutput StepOutput\n\t\t\tvar four int\n\n\t\t\terr := ctx.ParentOutput(skipOnEvent, &skipOnEventOutput)\n\n\t\t\tif err != nil {\n\t\t\t\tfour = 0\n\t\t\t} else {\n\t\t\t\tfour = skipOnEventOutput.RandomNumber\n\t\t\t}\n\n\t\t\tvar leftBranchOutput StepOutput\n\t\t\tvar five int\n\n\t\t\terr = ctx.ParentOutput(leftBranch, leftBranchOutput)\n\t\t\tif err != nil {\n\t\t\t\tfive = 0\n\t\t\t} else {\n\t\t\t\tfive = leftBranchOutput.RandomNumber\n\t\t\t}\n\n\t\t\tvar rightBranchOutput StepOutput\n\t\t\tvar six int\n\n\t\t\terr = ctx.ParentOutput(rightBranch, rightBranchOutput)\n\t\t\tif err != nil {\n\t\t\t\tsix = 0\n\t\t\t} else {\n\t\t\t\tsix = rightBranchOutput.RandomNumber\n\t\t\t}\n\n\t\t\treturn &RandomSum{\n\t\t\t\tSum: startOutput.RandomNumber + waitForEventOutput.RandomNumber +\n\t\t\t\t\twaitForSleepOutput.RandomNumber + four + five + six,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\treturn wf\n}\n',
  'source': 'out/go/workflows/complex-conditions.go',
  'blocks': {
    'create_a_workflow': {
      'start': 41,
      'stop': 46
    },
    'add_base_task': {
      'start': 49,
      'stop': 58
    },
    'add_wait_for_sleep': {
      'start': 61,
      'stop': 72
    },
    'add_skip_on_event': {
      'start': 75,
      'stop': 87
    },
    'add_branching': {
      'start': 90,
      'stop': 114
    },
    'add_wait_for_event': {
      'start': 117,
      'stop': 131
    },
    'add_sum': {
      'start': 134,
      'stop': 197
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
