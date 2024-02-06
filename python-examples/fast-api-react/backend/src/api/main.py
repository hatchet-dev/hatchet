from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse

from .models import MessageRequest

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


def event_stream_generator(workflowRunId):
    stream = hatchet.client.listener.generator(workflowRunId, 'complete')

    # TODO hatchet stream class
    # __iter__ = stream.__iter__()

    for event in stream:
        data = json.dumps({
            "type": event.type,
            "payload": event.payload,
            "workflowRunId": event.workflowRunId
        })
        # if something:
        #     stream.abort()
        print(data)
        yield "data: " + data + "\n\n"


@app.get("/stream/{messageId}")
async def stream(messageId: str):
    # message id -> workflowRunId
    workflowRunId = messageId
    # stream = hatchet.stream(workflowRunId)
    return StreamingResponse(event_stream_generator(workflowRunId), media_type='text/event-stream')
    # TODO how does client hangup


@app.post("/message")
def message(data: MessageRequest):
    print(data.model_dump())

    messageId = hatchet.client.admin.run_workflow("GenerateWorkflow", {
        "request": data.model_dump()
    })

    # save step message id -> workflowRunId

    return {"workflowRunId": messageId}


# TODO context is retry to not save data?

def start():
    """Launched with `poetry run start` at root level"""
    uvicorn.run("src.api.main:app", host="0.0.0.0", port=8000, reload=True)
