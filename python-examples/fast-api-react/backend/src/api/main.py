from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from hatchet_sdk import new_client
import uvicorn
from dotenv import load_dotenv

load_dotenv()

app = FastAPI()
hatchet = new_client()


origins = [
    "http://localhost:3000",
    "localhost:3000"
]


app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"]
)


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
