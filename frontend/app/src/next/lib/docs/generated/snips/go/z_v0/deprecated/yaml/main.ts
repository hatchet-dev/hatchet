import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    'package main\n\nimport (\n\t"context"\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserId   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\ntype actionInput struct {\n\tMessage string `json:"message"`\n}\n\ntype actionOut struct {\n\tMessage string `json:"message"`\n}\n\nfunc echo(ctx context.Context, input *actionInput) (result *actionOut, err error) {\n\treturn &actionOut{\n\t\tMessage: input.Message,\n\t}, nil\n}\n\nfunc object(ctx context.Context, input *userCreateEvent) error {\n\treturn nil\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New(\n\t\tclient.InitWorkflows(),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tworker, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\techoSvc := worker.NewService("echo")\n\n\terr = echoSvc.RegisterAction(echo)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = echoSvc.RegisterAction(object)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tch := cmdutils.InterruptChan()\n\n\tcleanup, err := worker.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf("error starting worker: %w", err))\n\t}\n\n\ttestEvent := userCreateEvent{\n\t\tUsername: "echo-test",\n\t\tUserId:   "1234",\n\t\tData: map[string]string{\n\t\t\t"test": "test",\n\t\t},\n\t}\n\n\ttime.Sleep(1 * time.Second)\n\n\t// push an event\n\terr = client.Event().Push(\n\t\tcontext.Background(),\n\t\t"user:create",\n\t\ttestEvent,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf("error cleaning up worker: %w", err))\n\t}\n}\n',
  source: 'out/go/z_v0/deprecated/yaml/main.go',
  blocks: {},
  highlights: {},
};

export default snippet;
