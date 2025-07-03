import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    'package main\n\nimport (\n\t"context"\n\t"fmt"\n\n\thatchet_client "github.com/hatchet-dev/hatchet/pkg/examples/quickstart/hatchet_client"\n\tworkflows "github.com/hatchet-dev/hatchet/pkg/examples/quickstart/workflows"\n)\n\nfunc main() {\n\thatchet, err := hatchet_client.HatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tsimple := workflows.FirstTask(hatchet)\n\n\tresult, err := simple.Run(context.Background(), workflows.SimpleInput{\n\t\tMessage: "Hello, World!",\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Println(\n\t\t"Finished running task, and got the transformed message! The transformed message is:",\n\t\tresult.ToLower.TransformedMessage,\n\t)\n}\n',
  source: 'out/go/quickstart/cmd/run/main.go',
  blocks: {},
  highlights: {},
};

export default snippet;
