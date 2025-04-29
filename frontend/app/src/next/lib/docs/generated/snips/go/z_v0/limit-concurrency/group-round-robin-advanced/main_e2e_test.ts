import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': '//go:build e2e\n\npackage main\n\nimport (\n\t\'context\'\n\t\'os\'\n\t\'os/signal\'\n\t\'testing\'\n\t\'time\'\n\n\t\'github.com/hatchet-dev/hatchet/internal/testutils\'\n)\n\nfunc TestAdvancedConcurrency(t *testing.T) {\n\tt.Skip(\'skipping concurency test for now\')\n\ttestutils.Prepare(t)\n\n\tctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)\n\tdefer cancel()\n\n\tch := make(chan os.Signal, 1)\n\tsignal.Notify(ch, os.Interrupt)\n\tgo func() {\n\t\t<-ctx.Done()\n\t\tch <- os.Interrupt\n\t}()\n\n\terr := run(ctx)\n\n\tif err != nil {\n\t\tt.Fatalf(\'/run() error = %v\', err)\n\t}\n\n}\n',
  'source': 'out/go/z_v0/limit-concurrency/group-round-robin-advanced/main_e2e_test.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
