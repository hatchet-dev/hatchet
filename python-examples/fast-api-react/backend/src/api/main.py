from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse

import time
from hatchet_sdk import Hatchet
import uvicorn
from dotenv import load_dotenv
import json

load_dotenv()

app = FastAPI()
hatchet = Hatchet()


origins = [
    "http://localhost:3001",
    "localhost:3001"
]


app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"]
)


async def event_stream_generator(workflowRunId):
    async for event in hatchet.client.listener.generator(workflowRunId):
        print(event.payload)
        yield "data: " + '{"test": "1"}' + "\n\n"


@app.get("/")
async def root():
    workflowRunId = hatchet.client.admin.run_workflow("ManualTriggerWorkflow", {
        "test": "test"
    })

    print(workflowRunId)

    return StreamingResponse(event_stream_generator(workflowRunId), media_type='text/event-stream')


def start():
    """Launched with `poetry run start` at root level"""
    uvicorn.run("src.api.main:app", host="0.0.0.0", port=8000, reload=True)
