import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'context\'\n\t\'fmt\'\n\t\'time\'\n\n\t\'github.com/joho/godotenv\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client\'\n\t\'github.com/hatchet-dev/hatchet/pkg/cmdutils\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype sampleEvent struct{}\n\ntype requeueInput struct{}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New(\n\t\tclient.InitWorkflows(),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tworker, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = worker.RegisterAction(\'requeue:requeue\', func(ctx context.Context, input *requeueInput) (result any, err error) {\n\t\treturn map[string]interface{}{}, nil\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tevent := sampleEvent{}\n\n\t// push an event\n\terr = client.Event().Push(\n\t\tcontext.Background(),\n\t\t\'example:event\',\n\t\tevent,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// wait to register the worker for 10 seconds, to let the requeuer kick in\n\ttime.Sleep(10 * time.Second)\n\tcleanup, err := worker.Start()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\tif err := cleanup(); err != nil {\n\t\t\t\tpanic(fmt.Errorf(\'error cleaning up: %w\', err))\n\t\t\t}\n\t\t\treturn\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n}\n',
  'source': 'out/go/z_v0/deprecated/requeue/main.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
