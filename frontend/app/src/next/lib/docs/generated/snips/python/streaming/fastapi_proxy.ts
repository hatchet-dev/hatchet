import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from typing import AsyncGenerator\n\nfrom fastapi import FastAPI\nfrom fastapi.responses import StreamingResponse\n\nfrom examples.streaming.worker import stream_task\nfrom hatchet_sdk import RunEventListener, StepRunEventType\n\n# > FastAPI Proxy\napp = FastAPI()\n\n\nasync def generate_stream(stream: RunEventListener) -> AsyncGenerator[str, None]:\n    async for chunk in stream:\n        if chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:\n            yield chunk.payload\n\n\n@app.get("/stream")\nasync def stream() -> StreamingResponse:\n    ref = await stream_task.aio_run_no_wait()\n\n    return StreamingResponse(generate_stream(ref.stream()), media_type="text/plain")\n\n\n\nif __name__ == "__main__":\n    import uvicorn\n\n    uvicorn.run(app, host="0.0.0.0", port=8000)\n',
  source: 'out/python/streaming/fastapi_proxy.py',
  blocks: {
    fastapi_proxy: {
      start: 10,
      stop: 25,
    },
  },
  highlights: {},
};

export default snippet;
