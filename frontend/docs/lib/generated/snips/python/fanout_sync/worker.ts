import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from datetime import timedelta\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions\n\nhatchet = Hatchet(debug=True)\n\n\nclass ParentInput(BaseModel):\n    n: int = 5\n\n\nclass ChildInput(BaseModel):\n    a: str\n\n\nsync_fanout_parent = hatchet.workflow(\n    name=\'SyncFanoutParent\', input_validator=ParentInput\n)\nsync_fanout_child = hatchet.workflow(name=\'SyncFanoutChild\', input_validator=ChildInput)\n\n\n@sync_fanout_parent.task(execution_timeout=timedelta(minutes=5))\ndef spawn(input: ParentInput, ctx: Context) -> dict[str, list[dict[str, Any]]]:\n    print(\'spawning child\')\n\n    results = sync_fanout_child.run_many(\n        [\n            sync_fanout_child.create_bulk_run_item(\n                input=ChildInput(a=str(i)),\n                key=f\'child{i}\',\n                options=TriggerWorkflowOptions(additional_metadata={\'hello\': \'earth\'}),\n            )\n            for i in range(input.n)\n        ],\n    )\n\n    print(f\'results {results}\')\n\n    return {\'results\': results}\n\n\n@sync_fanout_child.task()\ndef process(input: ChildInput, ctx: Context) -> dict[str, str]:\n    return {\'status\': \'success \' + input.a}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \'sync-fanout-worker\',\n        slots=40,\n        workflows=[sync_fanout_parent, sync_fanout_child],\n    )\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/fanout_sync/worker.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
