import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from typing import AsyncGenerator\n\nfrom fastapi import FastAPI\nfrom fastapi.responses import StreamingResponse\n\nfrom examples.streaming.worker import stream_task\nfrom hatchet_sdk import Hatchet\n\n# > FastAPI Proxy\nhatchet = Hatchet()\napp = FastAPI()\n\n\n@app.get("/stream")\nasync def stream() -> StreamingResponse:\n    ref = await stream_task.aio_run_no_wait()\n\n    return StreamingResponse(\n        hatchet.subscribe_to_stream(ref.workflow_run_id), media_type="text/plain"\n    )\n\n\n\nif __name__ == "__main__":\n    import uvicorn\n\n    uvicorn.run(app, host="0.0.0.0", port=8000)\n',
  source: 'out/python/streaming/fastapi_proxy.py',
  blocks: {
    fastapi_proxy: {
      start: 10,
      stop: 22,
    },
  },
  highlights: {},
};

export default snippet;
