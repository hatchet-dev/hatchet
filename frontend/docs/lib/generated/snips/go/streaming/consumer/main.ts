import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"log\"\n\n\t\"github.com/hatchet-dev/hatchet/examples/go/streaming/shared\"\n\tv1 \"github.com/hatchet-dev/hatchet/pkg/v1\"\n)\n\n// > Consume\nfunc main() {\n\thatchet, err := v1.NewHatchetClient()\n\tif err != nil {\n\t\tlog.Fatalf(\"Failed to create Hatchet client: %v\", err)\n\t}\n\n\tctx := context.Background()\n\n\tstreamingWorkflow := shared.StreamingWorkflow(hatchet)\n\n\tworkflowRun, err := streamingWorkflow.RunNoWait(ctx, shared.StreamTaskInput{})\n\tif err != nil {\n\t\tlog.Fatalf(\"Failed to run workflow: %v\", err)\n\t}\n\n\tid := workflowRun.RunId()\n\tstream, err := hatchet.Runs().SubscribeToStream(ctx, id)\n\tif err != nil {\n\t\tlog.Fatalf(\"Failed to subscribe to stream: %v\", err)\n\t}\n\n\tfor content := range stream {\n\t\tfmt.Print(content)\n\t}\n\n\tfmt.Println(\"\\nStreaming completed!\")\n}\n\n",
  "source": "out/go/streaming/consumer/main.go",
  "blocks": {
    "consume": {
      "start": 13,
      "stop": 40
    }
  },
  "highlights": {}
};

export default snippet;
