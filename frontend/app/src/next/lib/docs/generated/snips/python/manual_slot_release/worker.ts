import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "import time\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet()\n\n# > SlotRelease\n\nslot_release_workflow = hatchet.workflow(name='SlotReleaseWorkflow')\n\n\n@slot_release_workflow.task()\ndef step1(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print('RESOURCE INTENSIVE PROCESS')\n    time.sleep(10)\n\n    # 👀 Release the slot after the resource-intensive process, so that other steps can run\n    ctx.release_slot()\n\n    print('NON RESOURCE INTENSIVE PROCESS')\n    return {'status': 'success'}\n\n\n",
  source: 'out/python/manual_slot_release/worker.py',
  blocks: {
    slotrelease: {
      start: 8,
      stop: 23,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
