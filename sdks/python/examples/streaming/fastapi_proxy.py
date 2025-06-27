from typing import AsyncGenerator

from fastapi import FastAPI
from fastapi.responses import StreamingResponse

from examples.streaming.worker import stream_task
from hatchet_sdk import Hatchet

# > FastAPI Proxy
hatchet = Hatchet()
app = FastAPI()


@app.get("/stream")
async def stream() -> StreamingResponse:
    ref = await stream_task.aio_run_no_wait()

    return StreamingResponse(
        hatchet.subscribe_to_stream(ref.workflow_run_id), media_type="text/plain"
    )


# !!

if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
