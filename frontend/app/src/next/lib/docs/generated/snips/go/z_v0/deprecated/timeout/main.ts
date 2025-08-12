import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype sampleEvent struct{}\n\ntype timeoutInput struct{}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New(\n\t\tclient.InitWorkflows(),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tworker, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = worker.RegisterAction("timeout:timeout", func(ctx context.Context, input *timeoutInput) (result any, err error) {\n\t\t// wait for context done signal\n\t\ttimeStart := time.Now().UTC()\n\t\t<-ctx.Done()\n\t\tfmt.Println("context cancelled in ", time.Since(timeStart).Seconds(), " seconds")\n\n\t\treturn map[string]interface{}{}, nil\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tcleanup, err := worker.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error starting worker: %w", err))\n\t}\n\n\tevent := sampleEvent{}\n\n\t// push an event\n\terr = client.Event().Push(\n\t\tcontext.Background(),\n\t\t"user:create",\n\t\tevent,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\tif err := cleanup(); err != nil {\n\t\t\t\tpanic(fmt.Errorf("error cleaning up: %w", err))\n\t\t\t}\n\t\t\treturn\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
  source: 'out/go/z_v0/deprecated/timeout/main.go',
  blocks: {},
  highlights: {},
};

export default snippet;
