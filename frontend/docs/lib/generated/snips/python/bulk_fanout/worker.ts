import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from datetime import timedelta\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\nfrom hatchet_sdk.clients.admin import TriggerWorkflowOptions\n\nhatchet = Hatchet(debug=True)\n\n\nclass ParentInput(BaseModel):\n    n: int = 100\n\n\nclass ChildInput(BaseModel):\n    a: str\n\n\nbulk_parent_wf = hatchet.workflow(name=\'BulkFanoutParent\', input_validator=ParentInput)\nbulk_child_wf = hatchet.workflow(name=\'BulkFanoutChild\', input_validator=ChildInput)\n\n\n# > BulkFanoutParent\n@bulk_parent_wf.task(execution_timeout=timedelta(minutes=5))\nasync def spawn(input: ParentInput, ctx: Context) -> dict[str, list[dict[str, Any]]]:\n    # 👀 Create each workflow run to spawn\n    child_workflow_runs = [\n        bulk_child_wf.create_bulk_run_item(\n            input=ChildInput(a=str(i)),\n            key=f\'child{i}\',\n            options=TriggerWorkflowOptions(additional_metadata={\'hello\': \'earth\'}),\n        )\n        for i in range(input.n)\n    ]\n\n    # 👀 Run workflows in bulk to improve performance\n    spawn_results = await bulk_child_wf.aio_run_many(child_workflow_runs)\n\n    return {\'results\': spawn_results}\n\n\n\n\n@bulk_child_wf.task()\ndef process(input: ChildInput, ctx: Context) -> dict[str, str]:\n    print(f\'child process {input.a}\')\n    return {\'status\': \'success \' + input.a}\n\n\n@bulk_child_wf.task()\ndef process2(input: ChildInput, ctx: Context) -> dict[str, str]:\n    print(\'child process2\')\n    return {\'status2\': \'success\'}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'fanout-worker\', slots=40, workflows=[bulk_parent_wf, bulk_child_wf]\n    )\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/bulk_fanout/worker.py',
  'blocks': {
    'bulkfanoutparent': {
      'start': 25,
      'stop': 42
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
