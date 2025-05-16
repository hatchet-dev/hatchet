import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "import random\nimport time\nfrom datetime import timedelta\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\n\nclass StepOutput(BaseModel):\n    random_number: int\n\n\nclass RandomSum(BaseModel):\n    sum: int\n\n\nhatchet = Hatchet(debug=True)\n\ndag_workflow = hatchet.workflow(name='DAGWorkflow')\n\n\n@dag_workflow.task(execution_timeout=timedelta(seconds=5))\ndef step1(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n@dag_workflow.task(execution_timeout=timedelta(seconds=5))\nasync def step2(input: EmptyModel, ctx: Context) -> StepOutput:\n    return StepOutput(random_number=random.randint(1, 100))\n\n\n@dag_workflow.task(parents=[step1, step2])\nasync def step3(input: EmptyModel, ctx: Context) -> RandomSum:\n    one = ctx.task_output(step1).random_number\n    two = ctx.task_output(step2).random_number\n\n    return RandomSum(sum=one + two)\n\n\n@dag_workflow.task(parents=[step1, step3])\nasync def step4(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print(\n        'executed step4',\n        time.strftime('%H:%M:%S', time.localtime()),\n        input,\n        ctx.task_output(step1),\n        ctx.task_output(step3),\n    )\n    return {\n        'step4': 'step4',\n    }\n\n\ndef main() -> None:\n    worker = hatchet.worker('dag-worker', workflows=[dag_workflow])\n\n    worker.start()\n\n\nif __name__ == '__main__':\n    main()\n",
  source: 'out/python/dag/worker.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
