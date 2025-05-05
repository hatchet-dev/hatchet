import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package v1_workflows\n\nimport (\n\t\'errors\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client/create\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/factory\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/workflow\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype AlwaysFailsOutput struct {\n\tTransformedMessage string\n}\n\ntype OnFailureOutput struct {\n\tFailureRan bool\n}\n\ntype OnFailureSuccessResult struct {\n\tAlwaysFails AlwaysFailsOutput\n}\n\nfunc OnFailure(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[any, OnFailureSuccessResult] {\n\n\tsimple := factory.NewWorkflow[any, OnFailureSuccessResult](\n\t\tcreate.WorkflowCreateOpts[any]{\n\t\t\tName: \'on-failure\',\n\t\t},\n\t\thatchet,\n\t)\n\n\tsimple.Task(\n\t\tcreate.WorkflowTask[any, OnFailureSuccessResult]{\n\t\t\tName: \'AlwaysFails\',\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, _ any) (interface{}, error) {\n\t\t\treturn &AlwaysFailsOutput{\n\t\t\t\tTransformedMessage: \'always fails\',\n\t\t\t}, errors.New(\'always fails\')\n\t\t},\n\t)\n\n\tsimple.OnFailure(\n\t\tcreate.WorkflowOnFailureTask[any, OnFailureSuccessResult]{},\n\t\tfunc(ctx worker.HatchetContext, _ any) (interface{}, error) {\n\t\t\treturn &OnFailureOutput{\n\t\t\t\tFailureRan: true,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\treturn simple\n}\n',
  'source': 'out/go/workflows/on-failure.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
