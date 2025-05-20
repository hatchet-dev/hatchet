import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package main\n\nimport (\n\t\"fmt\"\n\n\t\"github.com/joho/godotenv\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client\"\n\t\"github.com/hatchet-dev/hatchet/pkg/cmdutils\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\ntype stepOneOutput struct {\n\tMessage string `json:\"message\"`\n}\n\n// > Backoff\n\n// ... normal function definition\nfunc StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\tif ctx.RetryCount() < 3 {\n\t\treturn nil, fmt.Errorf(\"failure\")\n\t}\n\n\treturn &stepOneOutput{\n\t\tMessage: \"Success!\",\n\t}, nil\n}\n\n// ,\n\nfunc main() {\n\t// ...\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ,\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tName:        \"retry-with-backoff-workflow\",\n\t\t\tOn:          worker.NoTrigger(),\n\t\t\tDescription: \"Demonstrates retry with exponential backoff.\",\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(StepOne).SetName(\"with-backoff\").\n\t\t\t\t\tSetRetries(10).\n\t\t\t\t\t// 👀 Backoff configuration\n\t\t\t\t\t// 👀 Maximum number of seconds to wait between retries\n\t\t\t\t\tSetRetryBackoffFactor(2.0).\n\t\t\t\t\t// 👀 Factor to increase the wait time between retries.\n\t\t\t\t\t// This sequence will be 2s, 4s, 8s, 16s, 32s, 60s... due to the maxSeconds limit\n\t\t\t\t\tSetRetryMaxBackoffSeconds(60),\n\t\t\t},\n\t\t},\n\t)\n\n\t// ...\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf(\"error cleaning up: %w\", err))\n\t}\n\n\t<-interruptCtx.Done()\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf(\"error cleaning up: %w\", err))\n\t}\n\n\t// ,\n}\n\n",
  "source": "out/go/z_v0/retries-with-backoff/main.go",
  "blocks": {
    "backoff": {
      "start": 18,
      "stop": 98
    }
  },
  "highlights": {}
};

export default snippet;
