import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'context\'\n\t\'fmt\'\n\t\'log\'\n\t\'time\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\nfunc run(done chan<- string, job worker.WorkflowJob) (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\'error creating client: %w\', err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\'error creating worker: %w\', err)\n\t}\n\n\terr = w.On(\n\t\tworker.Events(\'user:create:timeout\'),\n\t\t&job,\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\'error registering workflow: %w\', err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf(\'pushing event\')\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: \'echo-test\',\n\t\t\tUserID:   \'1234\',\n\t\t\tData: map[string]string{\n\t\t\t\t\'test\': \'test\',\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t\'user:create:timeout\',\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf(\'error pushing event: %w\', err))\n\t\t}\n\n\t\ttime.Sleep(20 * time.Second)\n\n\t\tdone <- \'done\'\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\'error starting worker: %w\', err)\n\t}\n\n\treturn cleanup, nil\n}\n',
  'source': 'out/go/z_v0/timeout/run.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
