from fastapi import FastAPI
from hatchet_sdk import new_client
import uvicorn
from dotenv import load_dotenv

load_dotenv()

app = FastAPI()
hatchet = new_client()


@app.get("/")
async def root():
    hatchet.event.create("poem:create", {"test": "test"})
    return {"message": "Hello World"}


@app.get("/fail")
async def root():
    hatchet.event.create("failure:create", {"test": "test"})
    return {"message": "Hello World"}


def start():
    """Launched with `poetry run start` at root level"""
    uvicorn.run("src.main:app", host="0.0.0.0", port=8000, reload=True)
