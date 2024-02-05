from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from hatchet_sdk import Hatchet
import uvicorn
from dotenv import load_dotenv

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


@app.get("/")
async def root():
    hatchet.event.push("question:create", {"test": "test"})
    return {"message": "Hello World"}


def start():
    """Launched with `poetry run start` at root level"""
    uvicorn.run("src.main:app", host="0.0.0.0", port=8000, reload=True)
