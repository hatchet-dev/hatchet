from fastapi import FastAPI
from hatchet_sdk import Hatchet, Context
import uvicorn

app = FastAPI()
hatchet = Hatchet()


@app.get("/")
async def root():
    return {"message": "Hello World"}


def start():
    """Launched with `poetry run start` at root level"""
    uvicorn.run("src.main:app", host="0.0.0.0", port=8000, reload=True)
