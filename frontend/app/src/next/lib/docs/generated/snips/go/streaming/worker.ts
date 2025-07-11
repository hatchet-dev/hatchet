import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    'package main\n\nimport (\n\t"log"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\tv1worker "github.com/hatchet-dev/hatchet/pkg/v1/worker"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype StreamTaskInput struct{}\n\ntype StreamTaskOutput struct {\n\tMessage string `json:"message"`\n}\n\n// > Streaming\nconst annaKarenina = `\nHappy families are all alike; every unhappy family is unhappy in its own way.\n\nEverything was in confusion in the Oblonskys\' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him.\n`\n\nfunc createChunks(content string, n int) []string {\n\tvar chunks []string\n\tfor i := 0; i < len(content); i += n {\n\t\tend := i + n\n\t\tif end > len(content) {\n\t\t\tend = len(content)\n\t\t}\n\t\tchunks = append(chunks, content[i:end])\n\t}\n\treturn chunks\n}\n\nfunc streamTask(ctx worker.HatchetContext, input StreamTaskInput) (*StreamTaskOutput, error) {\n\ttime.Sleep(2 * time.Second)\n\n\tchunks := createChunks(annaKarenina, 10)\n\n\tfor _, chunk := range chunks {\n\t\tctx.PutStream(chunk)\n\t\ttime.Sleep(200 * time.Millisecond)\n\t}\n\n\treturn &StreamTaskOutput{\n\t\tMessage: "Streaming completed",\n\t}, nil\n}\n\nfunc StreamingWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[StreamTaskInput, StreamTaskOutput] {\n\treturn factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "stream-example",\n\t\t},\n\t\tstreamTask,\n\t\thatchet,\n\t)\n}\n\n\nfunc main() {\n\thatchet, err := v1.NewHatchetClient()\n\tif err != nil {\n\t\tlog.Fatalf("Failed to create Hatchet client: %v", err)\n\t}\n\n\tstreamingWorkflow := StreamingWorkflow(hatchet)\n\n\tw, err := hatchet.Worker(v1worker.WorkerOpts{\n\t\tName: "streaming-worker",\n\t\tWorkflows: []workflow.WorkflowBase{\n\t\t\tstreamingWorkflow,\n\t\t},\n\t})\n\tif err != nil {\n\t\tlog.Fatalf("Failed to create worker: %v", err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.NewInterruptContext()\n\tdefer cancel()\n\n\tlog.Println("Starting streaming worker...")\n\terr = w.StartBlocking(interruptCtx)\n\tif err != nil {\n\t\tlog.Fatalf("Worker failed: %v", err)\n\t}\n}',
  source: 'out/go/streaming/worker.go',
  blocks: {
    streaming: {
      start: 23,
      stop: 65,
    },
  },
  highlights: {},
};

export default snippet;
