import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': '//go:build e2e\n\npackage main\n\nimport (\n\t\'context\'\n\t\'fmt\'\n\t\'testing\'\n\t\'time\'\n\n\t\'github.com/stretchr/testify/assert\'\n\n\t\'github.com/hatchet-dev/hatchet/internal/testutils\'\n)\n\nfunc TestProcedural(t *testing.T) {\n\ttestutils.Prepare(t)\n\n\tctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)\n\tdefer cancel()\n\n\tevents := make(chan string, 5*NUM_CHILDREN)\n\n\tcleanup, err := run(events)\n\tif err != nil {\n\t\tt.Fatalf(\'/run() error = %v\', err)\n\t}\n\n\tvar items []string\n\nouter:\n\tfor {\n\t\tselect {\n\t\tcase item := <-events:\n\t\t\titems = append(items, item)\n\t\tcase <-ctx.Done():\n\t\t\tbreak outer\n\t\t}\n\t}\n\n\texpected := []string{}\n\n\tfor i := 0; i < NUM_CHILDREN; i++ {\n\t\texpected = append(expected, fmt.Sprintf(\'child-%d-started\', i))\n\t\texpected = append(expected, fmt.Sprintf(\'child-%d-completed\', i))\n\t}\n\n\tassert.ElementsMatch(t, expected, items)\n\n\tif err := cleanup(); err != nil {\n\t\tt.Fatalf(\'cleanup() error = %v\', err)\n\t}\n}\n',
  'source': 'out/go/z_v0/procedural/main_e2e_test.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
