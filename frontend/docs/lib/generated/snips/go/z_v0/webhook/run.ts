import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'context\'\n\t\'errors\'\n\t\'fmt\'\n\t\'log\'\n\t\'net/http\'\n\t\'time\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\nfunc run(\n\tname string,\n\tw *worker.Worker,\n\tport string,\n\thandler func(w http.ResponseWriter, r *http.Request), c client.Client, workflow string, event string,\n) error {\n\t// create webserver to handle webhook requests\n\tmux := http.NewServeMux()\n\n\t// Register the HelloHandler to the /hello route\n\tmux.HandleFunc(\'/webhook\', handler)\n\n\t// Create a custom server\n\tserver := &http.Server{\n\t\tAddr:         \':\' + port,\n\t\tHandler:      mux,\n\t\tReadTimeout:  10 * time.Second,\n\t\tWriteTimeout: 10 * time.Second,\n\t\tIdleTimeout:  15 * time.Second,\n\t}\n\n\tdefer func(server *http.Server, ctx context.Context) {\n\t\terr := server.Shutdown(ctx)\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\t}(server, context.Background())\n\n\tgo func() {\n\t\tif err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {\n\t\t\tpanic(err)\n\t\t}\n\t}()\n\n\tsecret := \'secret\'\n\tif err := w.RegisterWebhook(worker.RegisterWebhookWorkerOpts{\n\t\tName:   \'test-\' + name,\n\t\tURL:    fmt.Sprintf(\'http://localhost:%s/webhook\', port),\n\t\tSecret: &secret,\n\t}); err != nil {\n\t\treturn fmt.Errorf(\'error setting up webhook: %w\', err)\n\t}\n\n\ttime.Sleep(30 * time.Second)\n\n\tlog.Printf(\'pushing event\')\n\n\ttestEvent := userCreateEvent{\n\t\tUsername: \'echo-test\',\n\t\tUserID:   \'1234\',\n\t\tData: map[string]string{\n\t\t\t\'test\': \'test\',\n\t\t},\n\t}\n\n\t// push an event\n\terr := c.Event().Push(\n\t\tcontext.Background(),\n\t\tevent,\n\t\ttestEvent,\n\t)\n\tif err != nil {\n\t\treturn fmt.Errorf(\'error pushing event: %w\', err)\n\t}\n\n\ttime.Sleep(5 * time.Second)\n\n\treturn nil\n}\n',
  'source': 'out/go/z_v0/webhook/run.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
