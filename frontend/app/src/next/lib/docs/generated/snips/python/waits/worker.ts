import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': '# > Create a workflow\n\nimport random\nfrom datetime import timedelta\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import (\n    Context,\n    EmptyModel,\n    Hatchet,\n    ParentCondition,\n    SleepCondition,\n    UserEventCondition,\n    or_,\n)\n\nhatchet = Hatchet(debug=True)\n\n\nclass StepOutput(BaseModel):\n    random_number: int\n\n\nclass RandomSum(BaseModel):\n    sum: int\n\n\ntask_condition_workflow = hatchet.workflow(name=\'TaskConditionWorkflow\')\n\n\n\n# > Add base task\n@task_condition_workflow.task()\ndef start(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n\n\n# > Add wait for sleep\n@task_condition_workflow.task(\n    parents=[start], wait_for=[SleepCondition(timedelta(seconds=10))]\n)\ndef wait_for_sleep(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n\n\n# > Add skip on event\n@task_condition_workflow.task(\n    parents=[start],\n    wait_for=[SleepCondition(timedelta(seconds=30))],\n    skip_if=[UserEventCondition(event_key=\'skip_on_event:skip\')],\n)\ndef skip_on_event(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n\n\n# > Add branching\n@task_condition_workflow.task(\n    parents=[wait_for_sleep],\n    skip_if=[\n        ParentCondition(\n            parent=wait_for_sleep,\n            expression=\'output.random_number > 50\',\n        )\n    ],\n)\ndef left_branch(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n@task_condition_workflow.task(\n    parents=[wait_for_sleep],\n    skip_if=[\n        ParentCondition(\n            parent=wait_for_sleep,\n            expression=\'output.random_number <= 50\',\n        )\n    ],\n)\ndef right_branch(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n\n\n# > Add wait for event\n@task_condition_workflow.task(\n    parents=[start],\n    wait_for=[\n        or_(\n            SleepCondition(duration=timedelta(minutes=1)),\n            UserEventCondition(event_key=\'wait_for_event:start\'),\n        )\n    ],\n)\ndef wait_for_event(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n\n\n# > Add sum\n@task_condition_workflow.task(\n    parents=[\n        start,\n        wait_for_sleep,\n        wait_for_event,\n        skip_on_event,\n        left_branch,\n        right_branch,\n    ],\n)\ndef sum(input: EmptyModel, ctx: Context) -> RandomSum:\n    one = ctx.task_output(start).random_number\n    two = ctx.task_output(wait_for_event).random_number\n    three = ctx.task_output(wait_for_sleep).random_number\n    four = (\n        ctx.task_output(skip_on_event).random_number\n        if not ctx.was_skipped(skip_on_event)\n        else 0\n    )\n\n    five = (\n        ctx.task_output(left_branch).random_number\n        if not ctx.was_skipped(left_branch)\n        else 0\n    )\n    six = (\n        ctx.task_output(right_branch).random_number\n        if not ctx.was_skipped(right_branch)\n        else 0\n    )\n\n    return RandomSum(sum=one + two + three + four + five + six)\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker(\'dag-worker\', workflows=[task_condition_workflow])\n\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/waits/worker.py',
  'blocks': {
    'create_a_workflow': {
      'start': 2,
      'stop': 30
    },
    'add_base_task': {
      'start': 34,
      'stop': 38
    },
    'add_wait_for_sleep': {
      'start': 42,
      'stop': 48
    },
    'add_skip_on_event': {
      'start': 52,
      'stop': 60
    },
    'add_branching': {
      'start': 64,
      'stop': 89
    },
    'add_wait_for_event': {
      'start': 93,
      'stop': 105
    },
    'add_sum': {
      'start': 109,
      'stop': 142
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
