import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package main\n\nimport (\n\t\"log\"\n\n\t\"github.com/hatchet-dev/hatchet/examples/go/streaming/shared\"\n\t\"github.com/hatchet-dev/hatchet/pkg/cmdutils\"\n\tv1 \"github.com/hatchet-dev/hatchet/pkg/v1\"\n\tv1worker \"github.com/hatchet-dev/hatchet/pkg/v1/worker\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/workflow\"\n)\n\nfunc main() {\n\thatchet, err := v1.NewHatchetClient()\n\tif err != nil {\n\t\tlog.Fatalf(\"Failed to create Hatchet client: %v\", err)\n\t}\n\n\tstreamingWorkflow := shared.StreamingWorkflow(hatchet)\n\n\tw, err := hatchet.Worker(v1worker.WorkerOpts{\n\t\tName: \"streaming-worker\",\n\t\tWorkflows: []workflow.WorkflowBase{\n\t\t\tstreamingWorkflow,\n\t\t},\n\t})\n\tif err != nil {\n\t\tlog.Fatalf(\"Failed to create worker: %v\", err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.NewInterruptContext()\n\tdefer cancel()\n\n\tlog.Println(\"Starting streaming worker...\")\n\n\tif err := w.StartBlocking(interruptCtx); err != nil {\n\t\tlog.Println(\"Worker failed:\", err)\n\t}\n}\n",
  "source": "out/go/streaming/worker/main.go",
  "blocks": {},
  "highlights": {}
};

export default snippet;
