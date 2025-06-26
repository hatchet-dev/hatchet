from typing import AsyncGenerator

from fastapi import FastAPI
from fastapi.responses import StreamingResponse

from examples.streaming.worker import stream_task
from hatchet_sdk import RunEventListener, StepRunEventType

# > FastAPI Proxy
app = FastAPI()


async def generate_stream(stream: RunEventListener) -> AsyncGenerator[str, None]:
    async for chunk in stream:
        if chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:
            yield chunk.payload


@app.get("/stream")
async def stream() -> StreamingResponse:
    ref = await stream_task.aio_run_no_wait()

    return StreamingResponse(generate_stream(ref.stream()), media_type="text/plain")


# !!

if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
