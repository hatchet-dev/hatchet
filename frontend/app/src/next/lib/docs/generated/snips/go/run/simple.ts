import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'context\'\n\t\'fmt\'\n\t\'sync\'\n\n\tv1_workflows \'github.com/hatchet-dev/hatchet/examples/go/workflows\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n\t\'github.com/joho/godotenv\'\n)\n\nfunc simple() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tctx := context.Background()\n\t// > Running a Task\n\tsimple := v1_workflows.Simple(hatchet)\n\tresult, err := simple.Run(ctx, v1_workflows.SimpleInput{\n\t\tMessage: \'Hello, World!\',\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Println(result.TransformedMessage)\n\n\t// > Running Multiple Tasks\n\tvar results []string\n\tvar resultsMutex sync.Mutex\n\tvar errs []error\n\tvar errsMutex sync.Mutex\n\n\twg := sync.WaitGroup{}\n\twg.Add(2)\n\n\tgo func() {\n\t\tdefer wg.Done()\n\t\tresult, err := simple.Run(ctx, v1_workflows.SimpleInput{\n\t\t\tMessage: \'Hello, World!\',\n\t\t})\n\n\t\tif err != nil {\n\t\t\terrsMutex.Lock()\n\t\t\terrs = append(errs, err)\n\t\t\terrsMutex.Unlock()\n\t\t\treturn\n\t\t}\n\n\t\tresultsMutex.Lock()\n\t\tresults = append(results, result.TransformedMessage)\n\t\tresultsMutex.Unlock()\n\t}()\n\n\tgo func() {\n\t\tdefer wg.Done()\n\t\tresult, err := simple.Run(ctx, v1_workflows.SimpleInput{\n\t\t\tMessage: \'Hello, Moon!\',\n\t\t})\n\n\t\tif err != nil {\n\t\t\terrsMutex.Lock()\n\t\t\terrs = append(errs, err)\n\t\t\terrsMutex.Unlock()\n\t\t\treturn\n\t\t}\n\n\t\tresultsMutex.Lock()\n\t\tresults = append(results, result.TransformedMessage)\n\t\tresultsMutex.Unlock()\n\t}()\n\n\twg.Wait()\n\n\t// > Running a Task Without Waiting\n\tsimple = v1_workflows.Simple(hatchet)\n\trunRef, err := simple.RunNoWait(ctx, v1_workflows.SimpleInput{\n\t\tMessage: \'Hello, World!\',\n\t})\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// The Run Ref Exposes an ID that can be used to wait for the task to complete\n\t// or check on the status of the task\n\trunId := runRef.RunId()\n\tfmt.Println(runId)\n\n\t// > Subscribing to results\n\t// finally, we can wait for the task to complete and get the result\n\tfinalResult, err := runRef.Result()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfmt.Println(finalResult)\n}\n',
  'source': 'out/go/run/simple.go',
  'blocks': {
    'running_a_task': {
      'start': 27,
      'stop': 36
    },
    'running_multiple_tasks': {
      'start': 39,
      'stop': 83
    },
    'running_a_task_without_waiting': {
      'start': 86,
      'stop': 98
    },
    'subscribing_to_results': {
      'start': 101,
      'stop': 108
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
