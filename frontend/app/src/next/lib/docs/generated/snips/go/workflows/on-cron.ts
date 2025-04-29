import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package v1_workflows\n\nimport (\n\t\'strings\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client/create\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/factory\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/workflow\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype OnCronInput struct {\n\tMessage string `json:\'Message\'`\n}\n\ntype JobResult struct {\n\tTransformedMessage string `json:\'TransformedMessage\'`\n}\n\ntype OnCronOutput struct {\n\tJob JobResult `json:\'job\'`\n}\n\n// > Workflow Definition Cron Trigger\nfunc OnCron(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[OnCronInput, OnCronOutput] {\n\t// Create a standalone task that transforms a message\n\tcronTask := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \'on-cron-task\',\n\t\t\t// 👀 add a cron expression\n\t\t\tOnCron: []string{\'0 0 * * *\'}, // Run every day at midnight\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input OnCronInput) (*OnCronOutput, error) {\n\t\t\treturn &OnCronOutput{\n\t\t\t\tJob: JobResult{\n\t\t\t\t\tTransformedMessage: strings.ToLower(input.Message),\n\t\t\t\t},\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn cronTask\n}\n\n\n',
  'source': 'out/go/workflows/on-cron.go',
  'blocks': {
    'workflow_definition_cron_trigger': {
      'start': 26,
      'stop': 46
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
