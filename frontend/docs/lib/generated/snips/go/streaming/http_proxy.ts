import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package main\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"log\"\n\t\"net/http\"\n\t\"time\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client/create\"\n\tv1 \"github.com/hatchet-dev/hatchet/pkg/v1\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/factory\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/workflow\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\ntype StreamTaskInput struct{}\n\ntype StreamTaskOutput struct {\n\tMessage string `json:\"message\"`\n}\n\nconst annaKarenina = `\nHappy families are all alike; every unhappy family is unhappy in its own way.\n\nEverything was in confusion in the Oblonskys' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him.\n`\n\nfunc createChunks(content string, n int) []string {\n\tvar chunks []string\n\tfor i := 0; i < len(content); i += n {\n\t\tend := i + n\n\t\tif end > len(content) {\n\t\t\tend = len(content)\n\t\t}\n\t\tchunks = append(chunks, content[i:end])\n\t}\n\treturn chunks\n}\n\nfunc streamTask(ctx worker.HatchetContext, input StreamTaskInput) (*StreamTaskOutput, error) {\n\ttime.Sleep(2 * time.Second)\n\n\tchunks := createChunks(annaKarenina, 10)\n\n\tfor _, chunk := range chunks {\n\t\tctx.PutStream(chunk)\n\t\ttime.Sleep(200 * time.Millisecond)\n\t}\n\n\treturn &StreamTaskOutput{\n\t\tMessage: \"Streaming completed\",\n\t}, nil\n}\n\nfunc StreamingWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[StreamTaskInput, StreamTaskOutput] {\n\treturn factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \"stream-example\",\n\t\t},\n\t\tstreamTask,\n\t\thatchet,\n\t)\n}\n\n// > HTTP Proxy\nfunc main() {\n\thatchet, err := v1.NewHatchetClient()\n\tif err != nil {\n\t\tlog.Fatalf(\"Failed to create Hatchet client: %v\", err)\n\t}\n\n\tstreamingWorkflow := StreamingWorkflow(hatchet)\n\n\thttp.HandleFunc(\"/stream\", func(w http.ResponseWriter, r *http.Request) {\n\t\tctx := context.Background()\n\n\t\tw.Header().Set(\"Content-Type\", \"text/plain\")\n\t\tw.Header().Set(\"Cache-Control\", \"no-cache\")\n\t\tw.Header().Set(\"Connection\", \"keep-alive\")\n\n\t\tworkflowRun, err := streamingWorkflow.RunNoWait(ctx, StreamTaskInput{})\n\t\tif err != nil {\n\t\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n\t\t\treturn\n\t\t}\n\n\t\tstream, err := hatchet.Runs().SubscribeToStream(ctx, workflowRun.RunId())\n\t\tif err != nil {\n\t\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n\t\t\treturn\n\t\t}\n\n\t\tflusher, _ := w.(http.Flusher)\n\t\tfor content := range stream {\n\t\t\tfmt.Fprint(w, content)\n\t\t\tif flusher != nil {\n\t\t\t\tflusher.Flush()\n\t\t\t}\n\t\t}\n\t})\n\n\tlog.Fatal(http.ListenAndServe(\":8000\", nil))\n}\n",
  "source": "out/go/streaming/http_proxy.go",
  "blocks": {
    "http_proxy": {
      "start": 67,
      "stop": 105
    }
  },
  "highlights": {}
};

export default snippet;
