import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package main\n\nimport (\n\t'context'\n\t'fmt'\n\n\tv1_workflows 'github.com/hatchet-dev/hatchet/examples/go/workflows'\n\tv1 'github.com/hatchet-dev/hatchet/pkg/v1'\n\t'github.com/joho/godotenv'\n)\n\nfunc bulk() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tctx := context.Background()\n\t// > Bulk Run Tasks\n\tsimple := v1_workflows.Simple(hatchet)\n\tbulkRunIds, err := simple.RunBulkNoWait(ctx, []v1_workflows.SimpleInput{\n\t\t{\n\t\t\tMessage: 'Hello, World!',\n\t\t},\n\t\t{\n\t\t\tMessage: 'Hello, Moon!',\n\t\t},\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Println(bulkRunIds)\n}\n",
  source: 'out/go/run/bulk.go',
  blocks: {
    bulk_run_tasks: {
      start: 26,
      stop: 40,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
