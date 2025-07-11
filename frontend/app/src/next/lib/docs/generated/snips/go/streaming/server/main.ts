import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"log"\n\t"net/http"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/examples/go/streaming/shared"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n)\n\n// > Server\nfunc main() {\n\thatchet, err := v1.NewHatchetClient()\n\tif err != nil {\n\t\tlog.Fatalf("Failed to create Hatchet client: %v", err)\n\t}\n\n\tstreamingWorkflow := shared.StreamingWorkflow(hatchet)\n\n\thttp.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {\n\t\tctx := context.Background()\n\n\t\tw.Header().Set("Content-Type", "text/plain")\n\t\tw.Header().Set("Cache-Control", "no-cache")\n\t\tw.Header().Set("Connection", "keep-alive")\n\n\t\tworkflowRun, err := streamingWorkflow.RunNoWait(ctx, shared.StreamTaskInput{})\n\t\tif err != nil {\n\t\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n\t\t\treturn\n\t\t}\n\n\t\tstream, err := hatchet.Runs().SubscribeToStream(ctx, workflowRun.RunId())\n\t\tif err != nil {\n\t\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n\t\t\treturn\n\t\t}\n\n\t\tflusher, _ := w.(http.Flusher)\n\t\tfor content := range stream {\n\t\t\tfmt.Fprint(w, content)\n\t\t\tif flusher != nil {\n\t\t\t\tflusher.Flush()\n\t\t\t}\n\t\t}\n\t})\n\n\tserver := &http.Server{\n\t\tAddr:         ":8000",\n\t\tReadTimeout:  5 * time.Second,\n\t\tWriteTimeout: 10 * time.Second,\n\t}\n\n\tif err := server.ListenAndServe(); err != nil {\n\t\tlog.Println("Failed to start server:", err)\n\t}\n}\n\n',
  source: 'out/go/streaming/server/main.go',
  blocks: {
    server: {
      start: 15,
      stop: 61,
    },
  },
  highlights: {},
};

export default snippet;
