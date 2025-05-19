import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'context\'\n\n\t\'github.com/google/uuid\'\n\t\'github.com/joho/godotenv\'\n\n\tv1_workflows \'github.com/hatchet-dev/hatchet/examples/go/workflows\'\n\t\'github.com/hatchet-dev/hatchet/pkg/client\'\n\t\'github.com/hatchet-dev/hatchet/pkg/client/rest\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n)\n\nfunc event() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\t// > Pushing an Event\n\terr = hatchet.Events().Push(\n\t\tcontext.Background(),\n\t\t\'simple-event:create\',\n\t\tv1_workflows.SimpleInput{\n\t\t\tMessage: \'Hello, World!\',\n\t\t},\n\t)\n\n\t// > Create a filter\n\tpayload := map[string]interface{}{\n\t\t\'main_character\':       \'Anna\',\n\t\t\'supporting_character\': \'Stiva\',\n\t\t\'location\':             \'Moscow\',\n\t}\n\n\t_, err = hatchet.Filters().Create(\n\t\tcontext.Background(),\n\t\trest.V1CreateFilterRequest{\n\t\t\tWorkflowId: uuid.New(),\n\t\t\tExpression: \'input.shouldSkip == false\',\n\t\t\tScope:      \'foobarbaz\',\n\t\t\tPayload:    &payload,\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// > Skip a run\n\tskipPayload := map[string]interface{}{\n\t\t\'shouldSkip\': true,\n\t}\n\tskipScope := \'foobarbaz\'\n\terr = hatchet.Events().Push(\n\t\tcontext.Background(),\n\t\t\'simple-event:create\',\n\t\tskipPayload,\n\t\tclient.WithFilterScope(&skipScope),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// > Trigger a run\n\ttriggerPayload := map[string]interface{}{\n\t\t\'shouldSkip\': false,\n\t}\n\ttriggerScope := \'foobarbaz\'\n\terr = hatchet.Events().Push(\n\t\tcontext.Background(),\n\t\t\'simple-event:create\',\n\t\ttriggerPayload,\n\t\tclient.WithFilterScope(&triggerScope),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n',
  'source': 'out/go/run/event.go',
  'blocks': {
    'pushing_an_event': {
      'start': 27,
      'stop': 33
    },
    'create_a_filter': {
      'start': 36,
      'stop': 50
    },
    'skip_a_run': {
      'start': 57,
      'stop': 66
    },
    'trigger_a_run': {
      'start': 73,
      'stop': 82
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
