import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    'package main\n\nimport (\n\t"fmt"\n\t"time"\n\n\t"github.com/joho/godotenv"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client"\n\t"github.com/hatchet-dev/hatchet/pkg/cmdutils"\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:"username"`\n\tUserID   string            `json:"user_id"`\n\tData     map[string]string `json:"data"`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\tif err := run(cmdutils.InterruptChan(), events); err != nil {\n\t\tpanic(err)\n\t}\n}\n\nfunc run(ch <-chan interface{}, events chan<- string) error {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error creating client: %w", err)\n\t}\n\n\ttime.Sleep(1 * time.Second)\n\n\t// trigger workflow\n\tworkflow, err := c.Admin().RunWorkflow(\n\t\t"post-user-update",\n\t\t&userCreateEvent{\n\t\t\tUsername: "echo-test",\n\t\t\tUserID:   "1234",\n\t\t\tData: map[string]string{\n\t\t\t\t"test": "test",\n\t\t\t},\n\t\t},\n\t\tclient.WithRunMetadata(map[string]interface{}{\n\t\t\t"hello": "world",\n\t\t}),\n\t)\n\n\tif err != nil {\n\t\treturn fmt.Errorf("error running workflow: %w", err)\n\t}\n\n\tfmt.Println("workflow run id:", workflow.WorkflowRunId())\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(ch)\n\tdefer cancel()\n\n\terr = c.Subscribe().On(interruptCtx, workflow.WorkflowRunId(), func(event client.WorkflowEvent) error {\n\t\tfmt.Println(event.EventPayload)\n\n\t\treturn nil\n\t})\n\n\treturn err\n}\n',
  source: 'out/go/z_v0/manual-trigger/trigger/main.go',
  blocks: {},
  highlights: {},
};

export default snippet;
