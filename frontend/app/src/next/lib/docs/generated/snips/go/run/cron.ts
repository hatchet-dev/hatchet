import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'context\'\n\t\'fmt\'\n\n\tv1_workflows \'github.com/hatchet-dev/hatchet/examples/go/workflows\'\n\t\'github.com/hatchet-dev/hatchet/pkg/client/rest\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n\t\'github.com/joho/godotenv\'\n)\n\nfunc cron() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\t// > Create\n\tsimple := v1_workflows.Simple(hatchet)\n\n\tctx := context.Background()\n\n\tresult, err := simple.Cron(\n\t\tctx,\n\t\t\'daily-run\',\n\t\t\'0 0 * * *\',\n\t\tv1_workflows.SimpleInput{\n\t\t\tMessage: \'Hello, World!\',\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// it may be useful to save the cron id for later\n\tfmt.Println(result.Metadata.Id)\n\n\t// > Delete\n\thatchet.Crons().Delete(ctx, result.Metadata.Id)\n\n\t// > List\n\tcrons, err := hatchet.Crons().List(ctx, rest.CronWorkflowListParams{\n\t\tAdditionalMetadata: &[]string{\'user:daily-run\'},\n\t})\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\tfmt.Println(crons)\n}\n',
  'source': 'out/go/run/cron.go',
  'blocks': {
    'create': {
      'start': 25,
      'stop': 43
    },
    'delete': {
      'start': 46,
      'stop': 46
    },
    'list': {
      'start': 49,
      'stop': 51
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
