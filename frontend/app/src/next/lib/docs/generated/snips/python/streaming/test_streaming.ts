import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from subprocess import Popen\nfrom typing import Any\n\nimport pytest\n\nfrom examples.streaming.worker import chunks, stream_task\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.listeners.run_event_listener import StepRunEventType\n\n\n@pytest.mark.parametrize(\n    "on_demand_worker",\n    [\n        (\n            ["poetry", "run", "python", "examples/streaming/worker.py", "--slots", "1"],\n            8008,\n        )\n    ],\n    indirect=True,\n)\n@pytest.mark.parametrize("execution_number", range(5))  # run test multiple times\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_streaming_ordering_and_completeness(\n    execution_number: int,\n    hatchet: Hatchet,\n    on_demand_worker: Popen[Any],\n) -> None:\n    ref = await stream_task.aio_run_no_wait()\n\n    ix = 0\n    anna_karenina = ""\n\n    async for chunk in ref.stream():\n        if chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:\n            assert chunks[ix] == chunk.payload\n            ix += 1\n            anna_karenina += chunk.payload\n\n    assert ix == len(chunks)\n    assert anna_karenina == "".join(chunks)\n\n    await ref.aio_result()\n',
  source: 'out/python/streaming/test_streaming.py',
  blocks: {},
  highlights: {},
};

export default snippet;
