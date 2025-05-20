import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\n# > Workflow Definition Cron Trigger\n# Adding a cron trigger to a workflow is as simple\n# as adding a `cron expression` to the `on_cron`\n# prop of the workflow definition\n\ncron_workflow = hatchet.workflow(name=\"CronWorkflow\", on_crons=[\"* * * * *\"])\n\n\n@cron_workflow.task()\ndef step1(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    return {\n        \"time\": \"step1\",\n    }\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker(\"test-worker\", slots=1, workflows=[cron_workflow])\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/cron/workflow-definition.py",
  "blocks": {
    "workflow_definition_cron_trigger": {
      "start": 7,
      "stop": 20
    }
  },
  "highlights": {}
};

export default snippet;
