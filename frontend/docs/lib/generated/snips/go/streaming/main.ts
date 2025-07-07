import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"log\"\n\n\tv1 \"github.com/hatchet-dev/hatchet/pkg/v1\"\n)\n\nfunc main() {\n\t// > Setup\n\n\thatchet, err := v1.NewHatchetClient()\n\tif err != nil {\n\t\tlog.Fatalf(\"Failed to create Hatchet client: %v\", err)\n\t}\n\n\tctx := context.Background()\n\n\n\t// > Consume\n\t// Create the streaming workflow\n\tstreamingWorkflow := StreamingWorkflow(hatchet)\n\n\t// Run the streaming workflow\n\tworkflowRun, err := streamingWorkflow.RunNoWait(ctx, StreamTaskInput{})\n\tif err != nil {\n\t\tlog.Fatalf(\"Failed to run workflow: %v\", err)\n\t}\n\n\tid := workflowRun.RunId()\n\n\t// Subscribe to the stream using the V1 subscribeToStream method\n\tstream, err := hatchet.Runs().SubscribeToStream(ctx, id)\n\tif err != nil {\n\t\tlog.Fatalf(\"Failed to subscribe to stream: %v\", err)\n\t}\n\n\tfor content := range stream {\n\t\tfmt.Print(content)\n\t}\n\n\n\tfmt.Println(\"\\nStreaming completed!\")\n}",
  "source": "out/go/streaming/main.go",
  "blocks": {
    "setup": {
      "start": 13,
      "stop": 20
    },
    "consume": {
      "start": 23,
      "stop": 43
    }
  },
  "highlights": {}
};

export default snippet;
