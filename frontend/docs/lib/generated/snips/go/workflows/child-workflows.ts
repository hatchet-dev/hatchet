import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package v1_workflows\n\nimport (\n\t\'github.com/hatchet-dev/hatchet/pkg/client/create\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/factory\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/workflow\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype ChildInput struct {\n\tN int `json:\'n\'`\n}\n\ntype ValueOutput struct {\n\tValue int `json:\'value\'`\n}\n\ntype ParentInput struct {\n\tN int `json:\'n\'`\n}\n\ntype SumOutput struct {\n\tResult int `json:\'result\'`\n}\n\nfunc Child(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ChildInput, ValueOutput] {\n\tchild := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \'child\',\n\t\t}, func(ctx worker.HatchetContext, input ChildInput) (*ValueOutput, error) {\n\t\t\treturn &ValueOutput{\n\t\t\t\tValue: input.N,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn child\n}\n\nfunc Parent(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ParentInput, SumOutput] {\n\n\tchild := Child(hatchet)\n\tparent := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \'parent\',\n\t\t}, func(ctx worker.HatchetContext, input ParentInput) (*SumOutput, error) {\n\n\t\t\tsum := 0\n\n\t\t\t// Launch child workflows in parallel\n\t\t\tresults := make([]*ValueOutput, 0, input.N)\n\t\t\tfor j := 0; j < input.N; j++ {\n\t\t\t\tresult, err := child.RunAsChild(ctx, ChildInput{N: j}, workflow.RunAsChildOpts{})\n\n\t\t\t\tif err != nil {\n\t\t\t\t\t// firstErr = err\n\t\t\t\t\treturn nil, err\n\t\t\t\t}\n\n\t\t\t\tresults = append(results, result)\n\n\t\t\t}\n\n\t\t\t// Sum results from all children\n\t\t\tfor _, result := range results {\n\t\t\t\tsum += result.Value\n\t\t\t}\n\n\t\t\treturn &SumOutput{\n\t\t\t\tResult: sum,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn parent\n}\n',
  'source': 'out/go/workflows/child-workflows.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
