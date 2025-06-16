import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import pytest\n\nfrom examples.streaming.worker import stream_task, chunks\nfrom hatchet_sdk.clients.listeners.run_event_listener import (\n    StepRunEventType,\n)\n\n\n@pytest.mark.parametrize("execution_number", range(1))\nasync def test_streaming_ordering_and_completeness(execution_number: int) -> None:\n    ref = await stream_task.aio_run_no_wait()\n\n    ix = 0\n\n    async for chunk in ref._wrr.stream():\n        if chunk.type != StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:\n            assert ix == len(chunks)\n            assert chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_COMPLETED\n\n        assert chunk.payload == chunks[ix], (\n            f"Expected chunk {ix} to be \'{chunks[ix]}\', but got \'{chunk}\' for execution {execution_number + 1}."\n        )\n\n        ix += 1\n',
  source: 'out/python/streaming/test_streaming.py',
  blocks: {},
  highlights: {},
};

export default snippet;
