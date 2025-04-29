import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "//go:build e2e\n\npackage main\n\nimport (\n\t'context'\n\t'testing'\n\t'time'\n\n\t'github.com/stretchr/testify/assert'\n\n\t'github.com/hatchet-dev/hatchet/internal/testutils'\n)\n\nfunc TestConcurrency(t *testing.T) {\n\tt.Skip('skipping concurency test for now')\n\n\ttestutils.Prepare(t)\n\n\tctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)\n\tdefer cancel()\n\n\tevents := make(chan string, 50)\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tt.Fatalf('/run() error = %v', err)\n\t}\n\n\tvar items []string\n\nouter:\n\tfor {\n\t\tselect {\n\t\tcase item := <-events:\n\t\t\titems = append(items, item)\n\t\t\tif len(items) > 2 {\n\t\t\t\tbreak outer\n\t\t\t}\n\t\tcase <-ctx.Done():\n\t\t\tbreak outer\n\t\t}\n\t}\n\n\tassert.Equal(t, []string{\n\t\t'step-one',\n\t\t'step-two',\n\t}, items)\n\n\tif err := cleanup(); err != nil {\n\t\tt.Fatalf('cleanup() error = %v', err)\n\t}\n\n}\n",
  source: 'out/go/z_v0/concurrency/main_e2e_test.go',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
