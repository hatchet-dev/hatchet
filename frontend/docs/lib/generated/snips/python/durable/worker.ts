import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import time\nfrom datetime import timedelta\nfrom uuid import uuid4\n\nfrom hatchet_sdk import (\n    Context,\n    DurableContext,\n    EmptyModel,\n    Hatchet,\n    SleepCondition,\n    UserEventCondition,\n    or_,\n)\n\nhatchet = Hatchet(debug=True)\n\n# > Create a durable workflow\ndurable_workflow = hatchet.workflow(name=\"DurableWorkflow\")\n\n\nephemeral_workflow = hatchet.workflow(name=\"EphemeralWorkflow\")\n\n\n# > Add durable task\nEVENT_KEY = \"durable-example:event\"\nSLEEP_TIME = 5\n\n\n@durable_workflow.task()\nasync def ephemeral_task(input: EmptyModel, ctx: Context) -> None:\n    print(\"Running non-durable task\")\n\n\n@durable_workflow.durable_task()\nasync def durable_task(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:\n    print(\"Waiting for sleep\")\n    await ctx.aio_sleep_for(duration=timedelta(seconds=SLEEP_TIME))\n    print(\"Sleep finished\")\n\n    print(\"Waiting for event\")\n    await ctx.aio_wait_for(\n        \"event\",\n        UserEventCondition(event_key=EVENT_KEY, expression=\"true\"),\n    )\n    print(\"Event received\")\n\n    return {\n        \"status\": \"success\",\n    }\n\n\n\n\n# > Add durable tasks that wait for or groups\n\n\n@durable_workflow.durable_task()\nasync def wait_for_or_group_1(\n    _i: EmptyModel, ctx: DurableContext\n) -> dict[str, str | int]:\n    start = time.time()\n    wait_result = await ctx.aio_wait_for(\n        uuid4().hex,\n        or_(\n            SleepCondition(timedelta(seconds=SLEEP_TIME)),\n            UserEventCondition(event_key=EVENT_KEY),\n        ),\n    )\n\n    key = list(wait_result.keys())[0]\n    event_id = list(wait_result[key].keys())[0]\n\n    return {\n        \"runtime\": int(time.time() - start),\n        \"key\": key,\n        \"event_id\": event_id,\n    }\n\n\n\n\n@durable_workflow.durable_task()\nasync def wait_for_or_group_2(\n    _i: EmptyModel, ctx: DurableContext\n) -> dict[str, str | int]:\n    start = time.time()\n    wait_result = await ctx.aio_wait_for(\n        uuid4().hex,\n        or_(\n            SleepCondition(timedelta(seconds=6 * SLEEP_TIME)),\n            UserEventCondition(event_key=EVENT_KEY),\n        ),\n    )\n\n    key = list(wait_result.keys())[0]\n    event_id = list(wait_result[key].keys())[0]\n\n    return {\n        \"runtime\": int(time.time() - start),\n        \"key\": key,\n        \"event_id\": event_id,\n    }\n\n\n@ephemeral_workflow.task()\ndef ephemeral_task_2(input: EmptyModel, ctx: Context) -> None:\n    print(\"Running non-durable task\")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"durable-worker\", workflows=[durable_workflow, ephemeral_workflow]\n    )\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/durable/worker.py",
  "blocks": {
    "create_a_durable_workflow": {
      "start": 18,
      "stop": 18
    },
    "add_durable_task": {
      "start": 25,
      "stop": 51
    },
    "add_durable_tasks_that_wait_for_or_groups": {
      "start": 55,
      "stop": 79
    }
  },
  "highlights": {}
};

export default snippet;
