import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package main\n\nimport (\n\t'context'\n\n\tv1_workflows 'github.com/hatchet-dev/hatchet/examples/go/workflows'\n\tv1 'github.com/hatchet-dev/hatchet/pkg/v1'\n\t'github.com/joho/godotenv'\n)\n\nfunc event() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\t// > Pushing an Event\n\terr = hatchet.Events().Push(\n\t\tcontext.Background(),\n\t\t'simple-event:create',\n\t\tv1_workflows.SimpleInput{\n\t\t\tMessage: 'Hello, World!',\n\t\t},\n\t\tnil,\n\t\tnil,\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n",
  source: 'out/go/run/event.go',
  blocks: {
    pushing_an_event: {
      start: 23,
      stop: 31,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
