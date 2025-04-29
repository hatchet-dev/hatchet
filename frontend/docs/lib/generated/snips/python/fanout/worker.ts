import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from datetime import timedelta\nfrom typing import Any\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions\n\nhatchet = Hatchet(debug=True)\n\n\n# > FanoutParent\nclass ParentInput(BaseModel):\n    n: int = 100\n\n\nclass ChildInput(BaseModel):\n    a: str\n\n\nparent_wf = hatchet.workflow(name=\'FanoutParent\', input_validator=ParentInput)\nchild_wf = hatchet.workflow(name=\'FanoutChild\', input_validator=ChildInput)\n\n\n@parent_wf.task(execution_timeout=timedelta(minutes=5))\nasync def spawn(input: ParentInput, ctx: Context) -> dict[str, Any]:\n    print(\'spawning child\')\n\n    result = await child_wf.aio_run_many(\n        [\n            child_wf.create_bulk_run_item(\n                input=ChildInput(a=str(i)),\n                options=TriggerWorkflowOptions(\n                    additional_metadata={\'hello\': \'earth\'}, key=f\'child{i}\'\n                ),\n            )\n            for i in range(input.n)\n        ]\n    )\n\n    print(f\'results {result}\')\n\n    return {\'results\': result}\n\n\n\n\n\n# > FanoutChild\n@child_wf.task()\ndef process(input: ChildInput, ctx: Context) -> dict[str, str]:\n    print(f\'child process {input.a}\')\n    return {\'status\': input.a}\n\n\n@child_wf.task(parents=[process])\ndef process2(input: ChildInput, ctx: Context) -> dict[str, str]:\n    process_output = ctx.task_output(process)\n    a = process_output[\'status\']\n\n    return {\'status2\': a + \'2\'}\n\n\n\n\nchild_wf.create_bulk_run_item()\n\n\ndef main() -> None:\n    worker = hatchet.worker(\'fanout-worker\', slots=40, workflows=[parent_wf, child_wf])\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/fanout/worker.py',
  'blocks': {
    'fanoutparent': {
      'start': 12,
      'stop': 44
    },
    'fanoutchild': {
      'start': 48,
      'stop': 61
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
